package provider

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func authzAllowResources(t *testing.T, realm string, uris []string) {
	t.Helper()
	base := os.Getenv("FERRISKEY_URL")
	authRealm := os.Getenv("FERRISKEY_REALM")

	form := url.Values{
		"grant_type": {"password"},
		"client_id":  {os.Getenv("FERRISKEY_CLIENT_ID")},
		"username":   {os.Getenv("FERRISKEY_USERNAME")},
		"password":   {os.Getenv("FERRISKEY_PASSWORD")},
	}
	tokResp, err := http.PostForm(base+"/realms/"+authRealm+"/protocol/openid-connect/token", form)
	if err != nil {
		t.Fatal(err)
	}
	defer tokResp.Body.Close()
	var tok struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(tokResp.Body).Decode(&tok); err != nil {
		t.Fatal(err)
	}

	body, _ := json.Marshal(map[string]any{"allowed_resources": uris})
	req, _ := http.NewRequest(http.MethodPut, base+"/realms/"+realm+"/settings", strings.NewReader(string(body)))
	req.Header.Set("Authorization", "Bearer "+tok.AccessToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("setting allowed_resources on %s: status %d", realm, resp.StatusCode)
	}
}

func TestAccAuthzResourceOwnersResource(t *testing.T) {
	realm := authzRealmName()
	base := fmt.Sprintf(`
resource "ferriskey_realm" "test" {
  name = %q
}

resource "ferriskey_client" "a" {
  realm         = ferriskey_realm.test.name
  client_id     = "owner-a"
  name          = "Owner A"
  client_type   = "confidential"
  redirect_uris = ["https://a.example.com/cb"]
}

resource "ferriskey_client" "b" {
  realm         = ferriskey_realm.test.name
  client_id     = "owner-b"
  name          = "Owner B"
  client_type   = "confidential"
  redirect_uris = ["https://b.example.com/cb"]
}
`, realm)

	owners := func(body string) string {
		return base + fmt.Sprintf(`
resource "ferriskey_resource_owners" "test" {
  realm  = ferriskey_realm.test.name
  owners = [%s]
}
`, body)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: base,
			},
			{
				PreConfig: func() {
					authzAllowResources(t, realm, []string{"https://mcp.example.com", "https://mcp2.example.com"})
				},
				Config: owners(`{ uri = "https://mcp.example.com", client_id = ferriskey_client.a.client_uuid }`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ferriskey_resource_owners.test", "owners.#", "1"),
					resource.TestCheckResourceAttr("ferriskey_resource_owners.test", "id", realm),
				),
			},
			{
				ResourceName:      "ferriskey_resource_owners.test",
				ImportState:       true,
				ImportStateId:     realm,
				ImportStateVerify: true,
			},
			{
				Config: owners(`
    { uri = "https://mcp.example.com", client_id = ferriskey_client.b.client_uuid },
    { uri = "https://mcp2.example.com", client_id = ferriskey_client.a.client_uuid },
  `),
				Check: resource.TestCheckResourceAttr("ferriskey_resource_owners.test", "owners.#", "2"),
			},
		},
	})
}
