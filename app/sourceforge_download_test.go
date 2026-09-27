package main

import "testing"

func TestSourceForgeHTMLRedirectKeepsExactArtifact(t *testing.T) {
	from := "https://sourceforge.net/projects/crystaldiskinfo/files/9.9.2/CrystalDiskInfo9_9_2.exe"
	page := []byte(`<meta http-equiv="refresh" content="5; url=https://downloads.sourceforge.net/project/crystaldiskinfo/9.9.2/CrystalDiskInfo9_9_2.exe?ts=token&amp;r=">`)
	if got := sourceForgeDownloadRedirect(from, page); got != "https://downloads.sourceforge.net/project/crystaldiskinfo/9.9.2/CrystalDiskInfo9_9_2.exe?ts=token&r=" {
		t.Fatalf("redirect=%s", got)
	}
	if sourceForgeDownloadRedirect(from+".other", page) != "" {
		t.Fatal("different artifact accepted")
	}
	if sourceForgeDownloadRedirect("https://other.example/projects/crystaldiskinfo/files/9.9.2/CrystalDiskInfo9_9_2.exe", page) != "" {
		t.Fatal("unrelated host accepted")
	}
}
