package main

// Physical runs established HKCU installations for these exact packages.
// Inno records an admin requirement even for /CURRENTUSER when the installer
// token is elevated (jrsoftware.org/ishelp/topic_admininstallmode.htm).
// Pin the declared user installer for download, HTTP proof and installation,
// then execute in the same ordinary-user context as the production GUI.
func vmPhysicalInstallScope(id string) string {
	switch id {
	case "OliverBetz.ExifTool":
		// Publisher manifest declares /CURRENTUSER. Use an ordinary token so
		// Inno does not record an admin requirement in this user installation.
		return "user"
	case "Microsoft.PowerToys":
		return "machine"
	case "Python.Python.3.13":
		// The publisher supports InstallAllUsers=1 and WinGet declares a
		// machine installer. The reduced-token user installer fails before
		// creating vendor logs on hosted runners; validate the machine variant
		// explicitly without claiming that the user variant has been repaired.
		return "machine"
	case "Telegram.TelegramDesktop", "GIMP.GIMP", "Greenshot.Greenshot", "WinSCP.WinSCP", "Microsoft.VisualStudioCode", "VSCodium.VSCodium", "WinMerge.WinMerge", "Jan.Jan",
		"Playnite.Playnite", "Ollama.Ollama", "HeidiSQL.HeidiSQL", "darktable.darktable", "Meltytech.Shotcut", "PostgreSQL.pgAdmin", "LiteXLTeam.LiteXL", "Brave.Brave", "Hibbiki.Chromium", "GLab.GLab", "TenacityTeam.Tenacity":
		return "user"
	}
	return ""
}
