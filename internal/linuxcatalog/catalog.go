package linuxcatalog

type Candidate struct {
	Name, Category           string
	Apt, Dnf, Pacman, Zypper []string
	Flatpak                  []string
}

var Candidates = []Candidate{
	{"Mozilla Firefox", "Böngészők", []string{"firefox-esr", "firefox"}, []string{"firefox"}, []string{"firefox"}, []string{"MozillaFirefox"}, []string{"org.mozilla.firefox"}},
	{"Chromium", "Böngészők", []string{"chromium", "chromium-browser"}, []string{"chromium"}, []string{"chromium"}, []string{"chromium"}, []string{"org.chromium.Chromium"}},
	{"Brave", "Böngészők", nil, nil, nil, nil, []string{"com.brave.Browser"}},
	{"Google Chrome", "Böngészők", []string{"google-chrome-stable"}, nil, nil, nil, []string{"com.google.Chrome"}},
	{"Vivaldi", "Böngészők", []string{"vivaldi-stable"}, nil, nil, nil, []string{"com.vivaldi.Vivaldi"}},
	{"Opera", "Böngészők", []string{"opera-stable"}, nil, nil, nil, []string{"com.opera.Opera"}},
	{"Tor Browser Launcher", "Böngészők", []string{"torbrowser-launcher"}, []string{"torbrowser-launcher"}, []string{"torbrowser-launcher"}, nil, []string{"com.github.micahflee.torbrowser-launcher"}},

	{"Discord", "Kommunikáció", nil, nil, nil, nil, []string{"com.discordapp.Discord"}},
	{"Zoom", "Kommunikáció", nil, nil, nil, nil, []string{"us.zoom.Zoom"}},
	{"Telegram Desktop", "Kommunikáció", []string{"telegram-desktop"}, []string{"telegram-desktop"}, []string{"telegram-desktop"}, []string{"telegram-desktop"}, []string{"org.telegram.desktop"}},
	{"Signal Desktop", "Kommunikáció", nil, nil, nil, nil, []string{"org.signal.Signal"}},
	{"Element", "Kommunikáció", nil, nil, nil, nil, []string{"im.riot.Riot"}},
	// Current publisher-verified ref: https://flathub.org/apps/org.mozilla.thunderbird
	{"Mozilla Thunderbird", "Kommunikáció", []string{"thunderbird"}, []string{"thunderbird"}, []string{"thunderbird"}, []string{"MozillaThunderbird"}, []string{"org.mozilla.thunderbird"}},
	{"Pidgin", "Kommunikáció", []string{"pidgin"}, []string{"pidgin"}, []string{"pidgin"}, []string{"pidgin"}, []string{"im.pidgin.Pidgin"}},

	{"VLC Media Player", "Média", []string{"vlc"}, []string{"vlc"}, []string{"vlc"}, []string{"vlc"}, []string{"org.videolan.VLC"}},
	{"Audacity", "Média", []string{"audacity"}, []string{"audacity"}, []string{"audacity"}, []string{"audacity"}, []string{"org.audacityteam.Audacity"}},
	{"HandBrake", "Média", []string{"handbrake"}, []string{"HandBrake-gui"}, []string{"handbrake"}, []string{"handbrake-gtk"}, []string{"fr.handbrake.ghb"}},
	{"OBS Studio", "Média", []string{"obs-studio"}, []string{"obs-studio"}, []string{"obs-studio"}, []string{"obs-studio"}, []string{"com.obsproject.Studio"}},
	{"Kdenlive", "Média", []string{"kdenlive"}, []string{"kdenlive"}, []string{"kdenlive"}, []string{"kdenlive"}, []string{"org.kde.kdenlive"}},
	{"Shotcut", "Média", []string{"shotcut"}, []string{"shotcut"}, []string{"shotcut"}, []string{"shotcut"}, []string{"org.shotcut.Shotcut"}},
	{"OpenShot", "Média", []string{"openshot-qt"}, []string{"openshot"}, []string{"openshot"}, []string{"openshot-qt"}, []string{"org.openshot.OpenShot"}},
	{"FFmpeg", "Média", []string{"ffmpeg"}, []string{"ffmpeg"}, []string{"ffmpeg"}, []string{"ffmpeg"}, nil},
	{"MuseScore", "Média", []string{"musescore3", "musescore"}, []string{"musescore"}, []string{"musescore"}, []string{"musescore"}, []string{"org.musescore.MuseScore"}},
	{"LMMS", "Média", []string{"lmms"}, []string{"lmms"}, []string{"lmms"}, []string{"lmms"}, []string{"io.lmms.LMMS"}},
	{"Ardour", "Média", []string{"ardour"}, []string{"ardour"}, []string{"ardour"}, []string{"ardour"}, []string{"org.ardour.Ardour"}},

	{"Krita", "Grafika", []string{"krita"}, []string{"krita"}, []string{"krita"}, []string{"krita"}, []string{"org.kde.krita"}},
	{"Blender", "Grafika", []string{"blender"}, []string{"blender"}, []string{"blender"}, []string{"blender"}, []string{"org.blender.Blender"}},
	{"GIMP", "Grafika", []string{"gimp"}, []string{"gimp"}, []string{"gimp"}, []string{"gimp"}, []string{"org.gimp.GIMP"}},
	{"Inkscape", "Grafika", []string{"inkscape"}, []string{"inkscape"}, []string{"inkscape"}, []string{"inkscape"}, []string{"org.inkscape.Inkscape"}},
	{"MyPaint", "Grafika", []string{"mypaint"}, []string{"mypaint"}, []string{"mypaint"}, []string{"mypaint"}, []string{"org.mypaint.MyPaint"}},
	{"Pinta", "Grafika", []string{"pinta"}, []string{"pinta"}, []string{"pinta"}, []string{"pinta"}, []string{"com.github.PintaProject.Pinta"}},
	{"darktable", "Grafika", []string{"darktable"}, []string{"darktable"}, []string{"darktable"}, []string{"darktable"}, []string{"org.darktable.Darktable"}},
	{"RawTherapee", "Grafika", []string{"rawtherapee"}, []string{"rawtherapee"}, []string{"rawtherapee"}, []string{"rawtherapee"}, []string{"com.rawtherapee.RawTherapee"}},
	{"digiKam", "Grafika", []string{"digikam"}, []string{"digikam"}, []string{"digikam"}, []string{"digikam"}, []string{"org.kde.digikam"}},
	{"FreeCAD", "Grafika / CAD", []string{"freecad"}, []string{"freecad"}, []string{"freecad"}, []string{"FreeCAD"}, []string{"org.freecad.FreeCAD"}},
	{"OpenSCAD", "Grafika / CAD", []string{"openscad"}, []string{"openscad"}, []string{"openscad"}, []string{"openscad"}, []string{"org.openscad.OpenSCAD"}},
	{"Scribus", "Grafika", []string{"scribus"}, []string{"scribus"}, []string{"scribus"}, []string{"scribus"}, []string{"net.scribus.Scribus"}},

	{"LibreOffice", "Dokumentumok", []string{"libreoffice"}, []string{"libreoffice"}, []string{"libreoffice-fresh", "libreoffice-still"}, []string{"libreoffice"}, []string{"org.libreoffice.LibreOffice"}},
	{"Okular", "Dokumentumok", []string{"okular"}, []string{"okular"}, []string{"okular"}, []string{"okular"}, []string{"org.kde.okular"}},
	{"Evince", "Dokumentumok", []string{"evince"}, []string{"evince"}, []string{"evince"}, []string{"evince"}, []string{"org.gnome.Evince"}},
	{"Xournal++", "Dokumentumok", []string{"xournalpp"}, []string{"xournalpp"}, []string{"xournalpp"}, []string{"xournalpp"}, []string{"com.github.xournalpp.xournalpp"}},
	{"Zotero", "Dokumentumok", nil, nil, nil, nil, []string{"org.zotero.Zotero"}},

	{"7-Zip / p7zip", "Tömörítés", []string{"7zip", "p7zip-full"}, []string{"p7zip", "p7zip-plugins"}, []string{"7zip", "p7zip"}, []string{"p7zip-full"}, nil},
	{"PeaZip", "Tömörítés", nil, nil, nil, nil, []string{"io.github.peazip.PeaZip"}},
	{"qBittorrent", "Torrent", []string{"qbittorrent"}, []string{"qbittorrent"}, []string{"qbittorrent"}, []string{"qbittorrent"}, []string{"org.qbittorrent.qBittorrent"}},
	{"Transmission", "Torrent", []string{"transmission-gtk"}, []string{"transmission-gtk"}, []string{"transmission-gtk"}, []string{"transmission-gtk"}, []string{"com.transmissionbt.Transmission"}},

	{"Git", "Fejlesztés", []string{"git"}, []string{"git"}, []string{"git"}, []string{"git"}, nil},
	{"Visual Studio Code", "Fejlesztés", []string{"code"}, []string{"code"}, []string{"code"}, nil, []string{"com.visualstudio.code"}},
	{"VSCodium", "Fejlesztés", []string{"codium"}, []string{"codium"}, []string{"vscodium"}, nil, []string{"com.vscodium.codium"}},
	{"Node.js", "Fejlesztés", []string{"nodejs"}, []string{"nodejs"}, []string{"nodejs"}, []string{"nodejs"}, nil},
	{"Python 3", "Fejlesztés", []string{"python3"}, []string{"python3"}, []string{"python"}, []string{"python3"}, nil},
	{"Go", "Fejlesztés", []string{"golang-go"}, []string{"golang"}, []string{"go"}, []string{"go"}, nil},
	{"Rust", "Fejlesztés", []string{"rustc", "cargo"}, []string{"rust", "cargo"}, []string{"rust"}, []string{"rust"}, nil},
	{"CMake", "Fejlesztés", []string{"cmake"}, []string{"cmake"}, []string{"cmake"}, []string{"cmake"}, nil},
	{"Eclipse IDE", "Fejlesztés", []string{"eclipse"}, []string{"eclipse"}, []string{"eclipse-java"}, nil, []string{"org.eclipse.Java"}},
	{"Apache NetBeans", "Fejlesztés", []string{"netbeans"}, []string{"netbeans"}, []string{"netbeans"}, []string{"netbeans"}, []string{"org.apache.netbeans"}},
	{"Android Studio", "Fejlesztés", nil, nil, nil, nil, []string{"com.google.AndroidStudio"}},
	{"IntelliJ IDEA Community", "Fejlesztés", nil, nil, nil, nil, []string{"com.jetbrains.IntelliJ-IDEA-Community"}},
	{"DBeaver Community", "Fejlesztés / Adatbázis", []string{"dbeaver-ce"}, []string{"dbeaver"}, []string{"dbeaver"}, nil, []string{"io.dbeaver.DBeaverCommunity"}},
	{"SQLite Browser", "Fejlesztés / Adatbázis", []string{"sqlitebrowser"}, []string{"sqlitebrowser"}, []string{"sqlitebrowser"}, []string{"sqlitebrowser"}, []string{"org.sqlitebrowser.sqlitebrowser"}},
	{"pgAdmin 4", "Fejlesztés / Adatbázis", []string{"pgadmin4-desktop", "pgadmin4"}, []string{"pgadmin4"}, nil, nil, nil},
	{"PostgreSQL", "Adatbázis", []string{"postgresql"}, []string{"postgresql-server"}, []string{"postgresql"}, []string{"postgresql-server"}, nil},
	{"MariaDB", "Adatbázis", []string{"mariadb-server"}, []string{"mariadb-server"}, []string{"mariadb"}, []string{"mariadb"}, nil},
	{"SQLite", "Adatbázis", []string{"sqlite3"}, []string{"sqlite"}, []string{"sqlite"}, []string{"sqlite3"}, nil},
	{"Neovim", "Fejlesztés", []string{"neovim"}, []string{"neovim"}, []string{"neovim"}, []string{"neovim"}, nil},
	{"Vim", "Fejlesztés", []string{"vim"}, []string{"vim-enhanced"}, []string{"vim"}, []string{"vim"}, nil},
	{"Emacs", "Fejlesztés", []string{"emacs"}, []string{"emacs"}, []string{"emacs"}, []string{"emacs"}, nil},
	{"Geany", "Fejlesztés", []string{"geany"}, []string{"geany"}, []string{"geany"}, []string{"geany"}, []string{"org.geany.Geany"}},

	{"Steam", "Játék", []string{"steam-installer", "steam"}, []string{"steam"}, []string{"steam"}, []string{"steam"}, []string{"com.valvesoftware.Steam"}},
	{"Heroic Games Launcher", "Játék", nil, nil, nil, nil, []string{"com.heroicgameslauncher.hgl"}},
	{"Lutris", "Játék", []string{"lutris"}, []string{"lutris"}, []string{"lutris"}, []string{"lutris"}, []string{"net.lutris.Lutris"}},
	{"Prism Launcher", "Játék", []string{"prismlauncher"}, []string{"prismlauncher"}, []string{"prismlauncher"}, nil, []string{"org.prismlauncher.PrismLauncher"}},
	{"itch.io App", "Játék", nil, nil, nil, nil, []string{"io.itch.itch"}},

	{"Wireshark", "Hálózat / IT", []string{"wireshark"}, []string{"wireshark"}, []string{"wireshark-qt"}, []string{"wireshark"}, []string{"org.wireshark.Wireshark"}},
	{"Nmap", "Hálózat / IT", []string{"nmap"}, []string{"nmap"}, []string{"nmap"}, []string{"nmap"}, nil},
	{"PuTTY", "Hálózat / IT", []string{"putty"}, []string{"putty"}, []string{"putty"}, []string{"putty"}, nil},
	{"FileZilla", "Hálózat / IT", []string{"filezilla"}, []string{"filezilla"}, []string{"filezilla"}, []string{"filezilla"}, []string{"org.filezillaproject.Filezilla"}},
	{"Remmina", "Hálózat / IT", []string{"remmina"}, []string{"remmina"}, []string{"remmina"}, []string{"remmina"}, []string{"org.remmina.Remmina"}},
	{"WireGuard Tools", "Hálózat / IT", []string{"wireguard-tools"}, []string{"wireguard-tools"}, []string{"wireguard-tools"}, []string{"wireguard-tools"}, nil},
	{"OpenVPN", "Hálózat / IT", []string{"openvpn"}, []string{"openvpn"}, []string{"openvpn"}, []string{"openvpn"}, nil},
	{"iperf3", "Hálózat / IT", []string{"iperf3"}, []string{"iperf3"}, []string{"iperf3"}, []string{"iperf3"}, nil},
	{"GNS3", "Hálózat / IT", []string{"gns3-gui"}, nil, nil, nil, nil},

	{"VirtualBox", "Virtualizáció", []string{"virtualbox"}, []string{"VirtualBox"}, []string{"virtualbox"}, []string{"virtualbox"}, nil},
	{"QEMU", "Virtualizáció", []string{"qemu-system"}, []string{"qemu-system-x86"}, []string{"qemu-full"}, []string{"qemu"}, nil},
	{"virt-manager", "Virtualizáció", []string{"virt-manager"}, []string{"virt-manager"}, []string{"virt-manager"}, []string{"virt-manager"}, []string{"org.virt_manager.virt-manager"}},
	{"GNOME Boxes", "Virtualizáció", []string{"gnome-boxes"}, []string{"gnome-boxes"}, []string{"gnome-boxes"}, []string{"gnome-boxes"}, []string{"org.gnome.Boxes"}},
	{"Podman", "Virtualizáció", []string{"podman"}, []string{"podman"}, []string{"podman"}, []string{"podman"}, nil},

	{"Syncthing", "Felhő / Szinkron", []string{"syncthing"}, []string{"syncthing"}, []string{"syncthing"}, []string{"syncthing"}, nil},
	{"rclone", "Felhő / Szinkron", []string{"rclone"}, []string{"rclone"}, []string{"rclone"}, []string{"rclone"}, nil},
	{"Nextcloud Desktop", "Felhő / Szinkron", []string{"nextcloud-desktop"}, []string{"nextcloud-client"}, []string{"nextcloud-client"}, []string{"nextcloud-desktop"}, []string{"com.nextcloud.desktopclient.nextcloud"}},

	{"KeePassXC", "Biztonság", []string{"keepassxc"}, []string{"keepassxc"}, []string{"keepassxc"}, []string{"keepassxc"}, []string{"org.keepassxc.KeePassXC"}},
	{"ClamAV", "Biztonság", []string{"clamav"}, []string{"clamav"}, []string{"clamav"}, []string{"clamav"}, nil},
	{"VeraCrypt", "Biztonság", []string{"veracrypt"}, nil, nil, nil, nil},

	{"Ollama", "Lokális AI", nil, nil, nil, nil, nil},
}
