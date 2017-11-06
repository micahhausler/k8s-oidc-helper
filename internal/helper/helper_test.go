package helper

import (
	"path/filepath"
	"testing"
)

func TestReadConfig(t *testing.T) {
	cases := []struct {
		file   string
		id     string
		secret string
	}{
		{
			"client_secret.json",
			"b6bf7f7e664e-2162b3fd698607beb27be0f7cb300005.apps.googleusercontent.com",
			"087e97f35e04bed75038f739",
		},
	}

	for _, c := range cases {
		path := filepath.Join("test-fixtures", c.file)

		got, err := ReadConfig(path)
		if err != nil {
			t.Errorf("Failed to load %s: %q", path, err)
			continue
		}

		if got.ClientID != c.id {
			t.Errorf("Got wrong client ID! Expected %s, got %s", c.id, got.ClientID)
		}
		if got.ClientSecret != c.secret {
			t.Errorf("Got wrong client secret! Expected %s, got %s", c.secret, got.ClientSecret)
		}
	}

}
