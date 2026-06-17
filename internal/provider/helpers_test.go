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
