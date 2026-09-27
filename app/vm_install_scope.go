package main

// Physical runs established HKCU installations for these exact packages.
// Inno records an admin requirement even for /CURRENTUSER when the installer
// token is elevated (jrsoftware.org/ishelp/topic_admininstallmode.htm).
// Pin the declared user installer for download, HTTP proof and installation,
// then execute in the same ordinary-user context as the production GUI.
func vmObservedUserInstallScope(id string) string {
	switch id {
	case "Telegram.TelegramDesktop", "GIMP.GIMP", "Greenshot.Greenshot", "WinSCP.WinSCP", "Python.Python.3.13",
		"Playnite.Playnite", "Ollama.Ollama", "HeidiSQL.HeidiSQL", "ZedIndustries.Zed", "darktable.darktable", "Meltytech.Shotcut":
		return "user"
	}
	return ""
}
