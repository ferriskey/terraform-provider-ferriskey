package provider

import (
	"fmt"
	"strings"

	"github.com/ferriskey/terraform-provider-ferriskey/internal/client"
)

// providerClient extracts the configured *client.Client from the value the
// provider stored in ResourceData/DataSourceData. It returns a descriptive
// error string suitable for a diagnostic when the type is unexpected (which
// should only happen due to a programming error).
func providerClient(data any) (*client.Client, string) {
	if data == nil {
		// Configure has not run yet (e.g. during provider validation). The
		// caller should treat a nil client as "skip".
		return nil, ""
	}
	c, ok := data.(*client.Client)
	if !ok {
		return nil, fmt.Sprintf("Expected *client.Client, got %T. This is a bug in the provider.", data)
	}
	return c, ""
}

// realmScopedID is a composite Terraform ID of the form "{realm}/{uuid}". It is
// used by every realm-scoped resource so that `terraform import` carries enough
// information to locate the object (the API requires the realm in the path) and
// so the ID is stable and human-readable.
type realmScopedID struct {
	Realm string
	ID    string
}

func (r realmScopedID) String() string {
	return r.Realm + "/" + r.ID
}

// parseRealmScopedID parses "{realm}/{uuid}". The realm itself never contains a
// slash in FerrisKey, so a single split is unambiguous.
func parseRealmScopedID(s string) (realmScopedID, error) {
	parts := strings.SplitN(s, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return realmScopedID{}, fmt.Errorf(
			"invalid import ID %q: expected the format \"{realm}/{uuid}\", e.g. \"master/3a8c6128-...\"", s)
	}
	return realmScopedID{Realm: parts[0], ID: parts[1]}, nil
}
