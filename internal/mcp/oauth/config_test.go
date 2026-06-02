package oauth

import "testing"

func TestRedirectURLRequiresLoopbackHTTP(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr string
	}{
		{
			name:    "https rejected",
			config:  Config{RedirectURL: "https://127.0.0.1:14545/oauth/callback"},
			wantErr: "redirect_url must use http for the local callback",
		},
		{
			name:    "non loopback rejected",
			config:  Config{RedirectURL: "http://example.com:14545/oauth/callback"},
			wantErr: "redirect_url host must be localhost or a loopback address",
		},
		{
			name:    "port required",
			config:  Config{RedirectURL: "http://127.0.0.1/oauth/callback"},
			wantErr: "redirect_url must include a non-zero port",
		},
		{
			name:   "default is valid",
			config: Config{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := RedirectURL(tt.config)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("RedirectURL() error = %v, want nil", err)
				}

				return
			}

			if err == nil || err.Error() != tt.wantErr {
				t.Fatalf("RedirectURL() error = %v, want %q", err, tt.wantErr)
			}
		})
	}
}
