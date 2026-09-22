package linuxpkg

import "testing"

func TestHostedRunnerMirrorDownload(t *testing.T) {
	uri := "mirror+file:/etc/apt/apt-mirrors.txt/pool/universe/v/vlc/vlc_3.0.20-3build6_amd64.deb"
	got, err := ResolveAPTDownload(uri, func(path string) ([]byte, error) {
		if path != "/etc/apt/apt-mirrors.txt" {
			t.Fatalf("unexpected path %s", path)
		}
		return []byte("# configured mirrors\nhttps://archive.ubuntu.com/ubuntu/\tpriority:1\n"), nil
	})
	if err != nil || got != "https://archive.ubuntu.com/ubuntu/pool/universe/v/vlc/vlc_3.0.20-3build6_amd64.deb" {
		t.Fatalf("%s %v", got, err)
	}
}
