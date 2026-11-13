package webgo

import (
	"net/http"
	"testing"
	"time"
)

func TestConfig_LoadInvalid(t *testing.T) {
	tl := useTestLogger(t)

	cfg := &Config{}
	cfg.Load("")
	str := tl.String()
	want := "open : no such file or directoryunexpected end of JSON inputport number not provided or is invalid (should be between 0 - 65535)"
	got := str
	if got != want {
		t.Errorf(
			"Expected '%s', got '%s'",
			want,
			got,
		)
	}
	tl.Reset()
}

func TestConfig_LoadValid(t *testing.T) {
	t.Parallel()
	cfg := Config{}
	cfg.Load("tests/config.json")

	cfg.Port = "a"
	if cfg.Validate() != ErrInvalidPort {
		t.Error("Port validation failed")
	}
	cfg.Port = "65536"
	if cfg.Validate() != ErrInvalidPort {
		t.Error("Port validation failed")
	}
}

func TestConfig_Validate(t *testing.T) {
	t.Parallel()
	type fields struct {
		Host               string
		Port               string
		CertFile           string
		KeyFile            string
		HTTPSPort          string
		ReadTimeout        time.Duration
		WriteTimeout       time.Duration
		InsecureSkipVerify bool
		ShutdownTimeout    time.Duration
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "invalid port",
			fields: fields{
				Port: "-12",
			},
			wantErr: true,
		},
		{
			name: "valid port",
			fields: fields{
				Port: "9000",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Host:               tt.fields.Host,
				Port:               tt.fields.Port,
				CertFile:           tt.fields.CertFile,
				KeyFile:            tt.fields.KeyFile,
				HTTPSPort:          tt.fields.HTTPSPort,
				ReadTimeout:        tt.fields.ReadTimeout,
				WriteTimeout:       tt.fields.WriteTimeout,
				InsecureSkipVerify: tt.fields.InsecureSkipVerify,
				ShutdownTimeout:    tt.fields.ShutdownTimeout,
			}
			if err := cfg.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}

}

func TestSetupServerTimeouts(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Port:              "9000",
		ReadTimeout:       1 * time.Second,
		ReadHeaderTimeout: 2 * time.Second,
		WriteTimeout:      3 * time.Second,
		IdleTimeout:       4 * time.Second,
	}
	router := NewRouter(cfg)
	router.setupServer()

	for name, srv := range map[string]*http.Server{"http": router.httpServer, "https": router.httpsServer} {
		if srv.ReadTimeout != cfg.ReadTimeout ||
			srv.ReadHeaderTimeout != cfg.ReadHeaderTimeout ||
			srv.WriteTimeout != cfg.WriteTimeout ||
			srv.IdleTimeout != cfg.IdleTimeout {
			t.Errorf("%s server timeouts do not match config: %+v", name, srv)
		}
	}

	// timeouts are disabled unless set
	router = NewRouter(&Config{Port: "9000"})
	router.setupServer()
	for name, srv := range map[string]*http.Server{"http": router.httpServer, "https": router.httpsServer} {
		if srv.ReadHeaderTimeout != 0 || srv.IdleTimeout != 0 {
			t.Errorf("%s server has unexpected default timeouts", name)
		}
	}
}
