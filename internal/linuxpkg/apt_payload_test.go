package linuxpkg

import "testing"

func TestPayloadMustBeDeclaredAndAvailable(t *testing.T) {
	available := map[string]bool{"postgresql-16": true, "emacs-gtk": true, "qemu-system-x86": true}
	for _, v := range [][3]string{{"postgresql", "  Depends: postgresql-16", "postgresql-16"}, {"emacs", " |Depends: emacs-gtk", "emacs-gtk"}, {"qemu-system", "  Depends: qemu-system-x86", "qemu-system-x86"}, {"postgresql", "  Suggests: postgresql-16", "postgresql"}, {"postgresql", "  Depends: postgresql-99", "postgresql"}, {"arbitrary", "  Depends: emacs-gtk", "arbitrary"}} {
		if got := APTPayloadPackage(v[0], v[1], available); got != v[2] {
			t.Fatalf("%v: %s", v, got)
		}
	}
}
