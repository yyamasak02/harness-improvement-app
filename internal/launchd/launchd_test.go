package launchd

import (
	"strings"
	"testing"
)

func TestPlist(t *testing.T) {
	s, err := Plist(Options{Label: "demo.label", Program: []string{"/usr/bin/open", "-a", "A&B <app>"}, Hour: 21, Minute: 5})
	if err != nil {
		t.Fatal(err)
	}
	for _, w := range []string{"<string>demo.label</string>", "<string>/usr/bin/open</string>", "A&amp;B &lt;app&gt;",
		"<key>Hour</key>\n    <integer>21</integer>", "<integer>5</integer>", "StartCalendarInterval"} {
		if !strings.Contains(s, w) {
			t.Errorf("%q が無い:\n%s", w, s)
		}
	}
}

func TestPlistInvalid(t *testing.T) {
	for _, o := range []Options{{Program: []string{"x"}}, {Label: "l"}, {Label: "l", Program: []string{"x"}, Hour: 24}, {Label: "l", Program: []string{"x"}, Minute: -1}} {
		if _, err := Plist(o); err == nil {
			t.Errorf("%+v", o)
		}
	}
}
