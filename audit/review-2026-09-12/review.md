# Projektáttekintés – 2026-09-12

Eredmény: **BLOCKED_ENVIRONMENT**. A csatolt feladat biztonsági előfeltétele alapján a végrehajtás megállt. Nem történt telepítés, eltávolítás, fizikai teszt vagy sikeres bizonyíték előállítása. Ez előzetes forráskód-áttekintés, nem teljes audit és nem kész implementáció.

## Ellenőrzött problémák

1. **Kritikus: a fizikai futtató nem bizonyítja a VM izolációját.** `test/windows-vm/run-one.ps1` automatikusan beállítja a tesztjelzőt és helyi VM-azonosítót. `app/vm_test_windows.go:runVMTestFromArgs` környezeti változókat, kulcsot és commitet ellenőriz, de nem igazol Windows 11-et, eldobható VM-et vagy tiszta snapshotot. A `run-range.ps1` ugyanazon gépen léptet programonként; az azonosító átírása nem állítja vissza a VM-et.
2. **Magas: tetszőleges munkakönyvtár törölhető.** `runVMTestOne` a CLI-ből kapott `workRoot` értékre `os.RemoveAll` hívást végez, célkönyvtár-korlátozás nélkül. Téves argumentum felhasználói adatot veszélyeztethet.
3. **Magas: megszakítás közben nincs tartós fázisonkénti checkpoint.** A JSON csak a teljes `runVMTestOne`/`resumeVMTestOne` visszatérése után íródik. Telepítés közbeni folyamatleállás után a már telepített program saját tesztbeli eredete elveszhet. A resume csak reboot státuszt fogad el.
4. **Magas: eltávolítás utáni reboot rossz fázisba tér vissza.** Mindkét reboot ugyanazt a státuszt kapja, a resume azonban először telepítettséget követel. Egy újraindításkor befejeződött sikeres eltávolítás így telepítésellenőrzési hibát eredményezhet.
5. **Magas: a bizonyíték aláírása nem fedi a teljes életciklus-adatot.** `internal/releaseproof/releaseproof.go:EvidenceStatement` a fő azonosítókat és ellenőrzési boolokat tartalmazza, de a részletes parancsokat, kilépési kódokat és detektálási rekordokat nem. Ezek kézi módosítását az adott HMAC önmagában nem mutatja ki.
6. **Közepes: a terv nincs saját katalógus-ujjlenyomattal ellátva.** `cmd/campaign-plan/main.go` dinamikusan generál indexelt tervet és státuszokat, de a terv struktúrájában nincs commit vagy fingerprint. Külön fingerprint-függvény már létezik a releaseproof csomagban.

## Feltérképezett elemek

- Katalógus/profil/licenc: `internal/catalog/{catalog,profile,license,audit}.go`; app-katalógus: `app/catalog.go`.
- Production feloldás és telepítés: `app/main_windows.go`, többek között `loadPackageData`, `startInstall`, `verifyProgramInstalled`.
- Registry és scope-felderítés: ugyanitt `applyInstalledPackages`, `listInstalledPackagesForScope`, `resolveInstalledPackageDetailed`, `resolveRegistryForInstalled`.
- Eltávolítás: `uninstallInContext`, `runRegisteredUninstaller`, `runWingetUninstallScoped`; stratégia és parancsfeldolgozás az `app/uninstall_*` fájlokban.
- VM-életciklus: `app/vm_test_windows.go`; retry, timeout, UAC és futó folyamatok kezelése: `app/vm_campaign_windows.go`. A production feloldási és eltávolítási utak újrafelhasználása részben már megvan. A Store teszt jelenleg kihagyási státusszal tér vissza.
- Build: `cmd/release-build`, `scripts/release-gate.ps1`, `scripts/release-gate.sh`; Windows app/installer és külön `linux/main_linux.go` belépési pont.
- Release bizonyítékok: `internal/releaseproof`, `internal/releasegate`, `cmd/release-gate`, `cmd/evidence-check`; adversarial és egységtesztek már vannak, ebben a környezetben nem futottak.

## Környezet és módosítások

A részletes aktuális környezet- és baseline-állapot az `environment-and-baseline.json` fájlban található. A meglévő `test/windows-vm/environment_evidence.json` szeptember 6-i Linux-környezetre vonatkozik, így nem igazolja ezt a Windows-munkamenetet.

A Go nem érhető el a PATH-on, a szokásos telepítési helyen sincs. A Winget alias megtalálható, de a `winget --version` üres kimenettel és -1978335231 kilépési kóddal hibázott. A CIM operációsrendszer- és géptípus-lekérdezés hozzáférési hibát adott. A nem indított baseline parancsok kilépési kódja null, nem siker.

Csak ez a jelentés és a mellette lévő környezet/baseline JSON készült. A már módosított build-, audit-, release- és tesztfájlokat megőriztem. A teljes életciklus próbakészletes igazolása, javítások, dinamikus tervgenerátor futtatása és a Release Gate ellenpróbája még hátravan. Folytatáshoz igazoltan eldobható Windows 11 x64 VM, programonként visszaállítható snapshot, működő Go és Winget szükséges.
