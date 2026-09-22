package main

import "testing"

func TestWingetSingleSpaceColumnsFromRealSearch(t *testing.T) {
	output := "Name     Id                Version\r\n----------------------------------\r\nPlaynite Playnite.Playnite 10.58\r\n"
	rows := wingetTableRows(output)
	if len(rows) != 1 || rows[0][0] != "Playnite" || rows[0][1] != "Playnite.Playnite" || rows[0][2] != "10.58" {
		t.Fatalf("lost exact Winget result: %#v", rows)
	}
}

func TestWingetProgressRedrawBeforeHeader(t *testing.T) {
	output := "\r   - \r                                                                                                                        \rName        Id                           Version\r\n------------------------------------------------------\r\nMediaMonkey VentisMedia.MediaMonkey.2024 2024.1.0.3113\r\nMediaMonkey VentisMedia.MediaMonkey.4    4.1\r\n"
	rows := wingetTableRows(output)
	if len(rows) != 2 || rows[0][0] != "MediaMonkey" || rows[0][1] != "VentisMedia.MediaMonkey.2024" || rows[1][1] != "VentisMedia.MediaMonkey.4" {
		t.Fatalf("progress output corrupted table identity: %#v", rows)
	}
}

func TestWingetPaddedNamesAndOptionalColumns(t *testing.T) {
	// Construct aligned data as Winget does, with the longest name using one gap.
	output := "Name           Id                   Version Available Source\n------------------------------------------------------------\nAndroid Studio Google.AndroidStudio 1.0     2.0       winget\n"
	rows := wingetTableRows(output)
	if len(rows) != 1 || rows[0][0] != "Android Studio" || rows[0][1] != "Google.AndroidStudio" || rows[0][2] != "1.0" || rows[0][4] != "winget" {
		t.Fatalf("bad column boundaries: %#v", rows)
	}
}
