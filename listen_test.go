package tool

import "testing"

func TestListenAddr(t *testing.T) {
	cases := []struct {
		name     string
		override string
		env      string
		fallback string
		want     string
	}{
		{"the role places the service", "", ":8112", ":8108", ":8112"},
		{"the default applies when the role sets nothing", "", "", ":8108", ":8108"},
		{"an explicit override wins", ":9000", ":8112", ":8108", ":9000"},
		{"a bare port from a role is normalized", "", "8112", ":8108", ":8112"},
		{"a bare port as a default is normalized", "", "", "8108", ":8108"},
		{"whitespace does not defeat placement", "", "  :8112  ", ":8108", ":8112"},
		{"an empty env is not a placement", "", "", ":8108", ":8108"},
		{"a host and port survives", "", "127.0.0.1:8112", ":8108", "127.0.0.1:8112"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(EnvListen, tc.env)
			if got := ListenAddr(tc.override, tc.fallback); got != tc.want {
				t.Errorf("ListenAddr(%q, %q) with %s=%q = %q, want %q",
					tc.override, tc.fallback, EnvListen, tc.env, got, tc.want)
			}
		})
	}
}

func TestListenAddrUnsetEnv(t *testing.T) {
	t.Setenv(EnvListen, "")
	if got := ListenAddr("", ":8108"); got != ":8108" {
		t.Errorf("ListenAddr = %q, want the fallback", got)
	}
}
