package provider

import "testing"

func TestParseRealmScopedID(t *testing.T) {
	cases := []struct {
		in        string
		wantRealm string
		wantID    string
		wantErr   bool
	}{
		{"master/3a8c6128-1111-2222-3333-444455556666", "master", "3a8c6128-1111-2222-3333-444455556666", false},
		{"my-realm/uuid", "my-realm", "uuid", false},
		{"noslash", "", "", true},
		{"/uuid", "", "", true},
		{"realm/", "", "", true},
		{"", "", "", true},
	}
	for _, tc := range cases {
		got, err := parseRealmScopedID(tc.in)
		if tc.wantErr {
			if err == nil {
				t.Errorf("parseRealmScopedID(%q): expected error, got %+v", tc.in, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseRealmScopedID(%q): unexpected error %v", tc.in, err)
			continue
		}
		if got.Realm != tc.wantRealm || got.ID != tc.wantID {
			t.Errorf("parseRealmScopedID(%q) = %+v, want realm=%q id=%q", tc.in, got, tc.wantRealm, tc.wantID)
		}
		if got.String() != tc.in {
			t.Errorf("round-trip String() = %q, want %q", got.String(), tc.in)
		}
	}
}

func TestParseCompositeID(t *testing.T) {
	got, err := parseCompositeID("master/c1/m1", "realm", "client_id", "id")
	if err != nil || len(got) != 3 || got[0] != "master" || got[1] != "c1" || got[2] != "m1" {
		t.Fatalf("unexpected result %v, %v", got, err)
	}
	for _, bad := range []string{"master/c1", "master//m1", "/c1/m1", ""} {
		if _, err := parseCompositeID(bad, "realm", "client_id", "id"); err == nil {
			t.Errorf("expected error for %q", bad)
		}
	}
}
