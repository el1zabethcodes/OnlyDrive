package doctor

import (
	"encoding/json"
	"strings"
	"testing"

	"gd/internal/config"
)

// withTempHome points GD_HOME at a throwaway directory for the test.
func withTempHome(t *testing.T) {
	t.Helper()
	t.Setenv("GD_HOME", t.TempDir())
}

// TestRunOAuthCheckCoversThreeBranches pins the three-way behavior of the
// oauth check: no accounts -> absent; own credentials -> OK; accounts on the
// shared client -> FAIL with the how-to text.
func TestRunOAuthCheckCoversThreeBranches(t *testing.T) {
	t.Run("no accounts", func(t *testing.T) {
		withTempHome(t)
		checks, err := Run(t.Context(), false)
		if err != nil {
			t.Fatal(err)
		}
		for _, c := range checks {
			if c.Name == "oauth" {
				t.Fatalf("oauth check must be absent without accounts, got %+v", c)
			}
		}
	})

	t.Run("own credentials from config", func(t *testing.T) {
		withTempHome(t)
		cfg, err := config.Load()
		if err != nil {
			t.Fatal(err)
		}
		cfg.Accounts = append(cfg.Accounts, config.Account{Name: "acc1", Email: "me@gmail.com", Remote: "gdrive-acc1"})
		cfg.OAuthApp = &config.OAuthApp{ClientID: "my-id", ClientSecret: "my-secret"}
		if err := cfg.Save(); err != nil {
			t.Fatal(err)
		}
		checks, err := Run(t.Context(), false)
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, c := range checks {
			if c.Name != "oauth" {
				continue
			}
			found = true
			if !c.OK {
				t.Fatalf("own credentials must pass, got %+v", c)
			}
			if !strings.Contains(c.Detail, "config") {
				t.Fatalf("detail should name the source, got %q", c.Detail)
			}
		}
		if !found {
			t.Fatal("oauth check missing")
		}
	})

	t.Run("accounts on shared client", func(t *testing.T) {
		withTempHome(t)
		cfg, err := config.Load()
		if err != nil {
			t.Fatal(err)
		}
		cfg.Accounts = append(cfg.Accounts, config.Account{Name: "acc1", Email: "me@gmail.com", Remote: "gdrive-acc1"})
		if err := cfg.Save(); err != nil {
			t.Fatal(err)
		}
		checks, err := Run(t.Context(), false)
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, c := range checks {
			if c.Name != "oauth" {
				continue
			}
			found = true
			if c.OK {
				t.Fatalf("shared client with accounts must fail, got %+v", c)
			}
			for _, want := range []string{"gd oauth set", "making-your-own-client-id"} {
				if !strings.Contains(c.Detail, want) {
					t.Fatalf("detail missing %q, got %q", want, c.Detail)
				}
			}
		}
		if !found {
			t.Fatal("oauth check missing")
		}
	})
}

// TestReportOf pins the overall verdict: OK only when every check passed.
func TestReportOf(t *testing.T) {
	allOK := []Check{
		{"rclone", true, "rclone v1.68.2"},
		{"daemon", true, "rclone rcd on 127.0.0.1:5572"},
	}
	if r := ReportOf(allOK); !r.OK {
		t.Fatalf("all checks passed, report must be ok, got %+v", r)
	}
	oneBad := append(allOK, Check{"accounts", false, "no Google accounts added (run: gd add)"})
	if r := ReportOf(oneBad); r.OK {
		t.Fatal("a failed check must fail the report")
	}
}

// TestReportJSON pins the machine-readable shape of `gd doctor --json`:
// tagged lowercase keys, values that survive a round-trip, indented output.
func TestReportJSON(t *testing.T) {
	report := ReportOf([]Check{
		{"rclone", true, "rclone v1.68.2"},
		{"accounts", false, "no Google accounts added (run: gd add)"},
	})
	out, err := report.JSON()
	if err != nil {
		t.Fatal(err)
	}

	var back struct {
		OK     bool    `json:"ok"`
		Checks []Check `json:"checks"`
	}
	if err := json.Unmarshal([]byte(out), &back); err != nil {
		t.Fatalf("not valid JSON: %v\n%s", err, out)
	}
	if back.OK {
		t.Fatal("ok must be false when a check failed")
	}
	if len(back.Checks) != 2 {
		t.Fatalf("want 2 checks after the round-trip, got %d", len(back.Checks))
	}
	got := back.Checks[1]
	if got.Name != "accounts" || got.OK || got.Detail != "no Google accounts added (run: gd add)" {
		t.Fatalf("check lost in the round-trip: %+v", got)
	}
	for _, key := range []string{`"ok"`, `"checks"`, `"name"`, `"detail"`} {
		if !strings.Contains(out, key) {
			t.Fatalf("output missing tagged key %s\n%s", key, out)
		}
	}
	if !strings.Contains(out, "\n  \"") {
		t.Fatalf("output must be indented for readability\n%s", out)
	}
}

// TestReportJSONFromRun makes sure a real Run's checks marshal too, not just
// hand-made ones, and the verdict matches the raw checks.
func TestReportJSONFromRun(t *testing.T) {
	withTempHome(t)
	checks, err := Run(t.Context(), false)
	if err != nil {
		t.Fatal(err)
	}
	out, err := ReportOf(checks).JSON()
	if err != nil {
		t.Fatal(err)
	}
	var back Report
	if err := json.Unmarshal([]byte(out), &back); err != nil {
		t.Fatalf("real report is not valid JSON: %v\n%s", err, out)
	}
	if len(back.Checks) == 0 {
		t.Fatal("real report must carry its checks")
	}
	anyBad := false
	for _, c := range checks {
		if !c.OK {
			anyBad = true
		}
	}
	if back.OK == anyBad {
		t.Fatalf("verdict mismatch: ok=%v but failed checks present=%v", back.OK, anyBad)
	}
}
