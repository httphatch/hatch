package session

import "testing"

func TestSessionID_String(t *testing.T) {
	id := SessionID{Project: "myapp", Name: "fix-auth"}
	if got := id.String(); got != "myapp/fix-auth" {
		t.Errorf("String() = %q, want %q", got, "myapp/fix-auth")
	}
}

func TestSessionID_QualifiedProject(t *testing.T) {
	id := SessionID{Project: "myapp", Name: "fix-auth"}
	if got := id.QualifiedProject(); got != "myapp~fix-auth" {
		t.Errorf("QualifiedProject() = %q, want %q", got, "myapp~fix-auth")
	}
}

func TestDomain(t *testing.T) {
	tests := []struct {
		name             string
		sessionName      string
		baseDomain       string
		serviceSubdomain string
		want             string
	}{
		{
			name:             "simple session subdomain",
			sessionName:      "fix-auth",
			baseDomain:       "myapp.test",
			serviceSubdomain: "",
			want:             "fix-auth.myapp.test",
		},
		{
			name:             "session with service subdomain",
			sessionName:      "fix-auth",
			baseDomain:       "myapp.test",
			serviceSubdomain: "api",
			want:             "fix-auth--api.myapp.test",
		},
		{
			name:             "auto-generated name",
			sessionName:      "s1",
			baseDomain:       "myapp.test",
			serviceSubdomain: "",
			want:             "s1.myapp.test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Domain(tt.sessionName, tt.baseDomain, tt.serviceSubdomain)
			if got != tt.want {
				t.Errorf("Domain() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSessionState_String(t *testing.T) {
	tests := []struct {
		state SessionState
		want  string
	}{
		{StateStarting, "starting"},
		{StateRunning, "running"},
		{StateStopping, "stopping"},
		{StateStopped, "stopped"},
		{SessionState(99), "unknown"},
	}

	for _, tt := range tests {
		if got := tt.state.String(); got != tt.want {
			t.Errorf("SessionState(%d).String() = %q, want %q", tt.state, got, tt.want)
		}
	}
}
