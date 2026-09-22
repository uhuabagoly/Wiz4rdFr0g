# Windows fizikai katalóguskampány

A `.github/workflows/windows-vm-package-test.yml` workflow a frissen generált katalógusterv összes `PHYSICAL_REQUIRED` elemét választja ki. Nincs kézzel rögzített ötprogramos lista. Minden kiválasztott elem külön, friss GitHub-hosted Windows VM-ben fut. A licenc vagy támogatási okból kizárt elemek a teljes tervben maradnak, és továbbra is blokkolhatják a teljes kiadást.

Indítás a módosításokat tartalmazó GitHub-ágon:

```sh
gh workflow run windows-vm-package-test.yml --repo uhuabagoly/Wiz4rdFr0g --ref AGNEV
```

Az `AGNEV` a feltöltött ág neve. Szükséges repository secret: `WIZ4RDFR0G_EVIDENCE_HMAC_KEY`, legalább 32 bájt. Ne commitold vagy logold az értékét. A workflow alapértelmezett ágon már létezik, de az adott módosított ágat fel kell tölteni a futtatáshoz.

A plan job telepíti a Linux GTK buildfüggőségeit, lefuttatja a unit/static ellenőrzéseket, előállítja a release artifactokat és a teljes Windows-tervet, majd ellenőrzi a kapu üres bizonyítékhalmazzal történő blokkolását. A package-test jobok a production executable-t használják és független telepítettség-ellenőrzést végeznek. Az aggregate az összes jogosult index aktuális, hiteles életciklusát megköveteli. Hiányzó vagy duplikált index nem elfogadható.

A `CAMPAIGN_PASS` kizárólag a jogosult Windows-kampány sikere. Nem jelenti a teljes Windows-katalógus, Linux-katalógus, USB-portabilitás vagy teljes release sikerét. A Linux fizikai kampányának implementációja még hiányzik. 256 jogosult elem felett a jelenlegi tervgenerátor leáll: további workflow-particionálás szükséges, ezt nem állítjuk kész shardingnak.

Artifactnevek kompatibilitásból még `pilot-build-RUN-ATTEMPT`, `wf-package-RUN-ATTEMPT-INDEX`, `pilot-report-RUN-ATTEMPT`. Tartalmuk: build manifest, baseline parancskimenetek, katalógusterv, kiválasztott indexek, VM-diagnosztika, aláírt életciklus-JSON és aggregálási log. A HMAC a teljes JSON-t is védi.

Megszakítás után új teljes lifecycle szükséges friss VM-en. A jelenlegi futás-/attempt-kötés miatt `Re-run all jobs` használandó; csak a failed jobok újrafuttatása nem támogatott. Reboot és előtelepített program nem eredményez PASS-t; előtelepített szoftvert a teszt nem távolít el. Windows Server runner nem igazol Windows 11 klienskompatibilitást.

Külön workflow: `portable-validation.yml` Windows/Linux fordítást, unit/static teszteket és kicsomagolt Linux bináris diagnosztikai indulást ellenőriz. Ez nem fizikai csomag-életciklus.
