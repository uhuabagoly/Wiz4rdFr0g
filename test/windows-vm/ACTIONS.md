# Ötprogramos GitHub Actions próba

Helyi állapot: **REMOTE_EXECUTION_REQUIRED**. Ebben a munkamenetben nincs Git remote, ezért nem történt push vagy Actions-indítás. Nincs új fizikai PASS. A helyi Go/Winget nem előfeltétel az előkészítéshez.

## Indítás a GitHub felületén

1. A lenti módosításokat a projekt többi szükséges forrásával együtt commitold és juttasd el a GitHub-repóba. Ne csak a YAML-fájlt másold át. A `workflow_dispatch` workflow-nak a repó alapértelmezett ágán is jelen kell lennie ahhoz, hogy a kézi indítás megjelenjen.
2. A repóban: **Settings → Secrets and variables → Actions → New repository secret**. Név: `WIZ4RDFR0G_EVIDENCE_HMAC_KEY`. Érték: legalább 32 karakteres, kriptográfiailag véletlen titok. Például PowerShell 7-ben `[Convert]::ToHexString([Security.Cryptography.RandomNumberGenerator]::GetBytes(32))` generál megfelelő értéket. A titkot ne commitold.
3. **Actions → Windows package install-uninstall VM test → Run workflow**. Válaszd a módosításokat tartalmazó ágat, majd indítsd el. Nincs start/count bemenet: ez a workflow szándékosan csak öt próbaelemet enged.
4. A `plan` job elvégzi a Go unit/static ellenőrzéseket, a Windows és Linux release buildet, előállítja a teljes katalógustervet és hashét, majd ellenőrzi, hogy az üres eredményhalmaz blokkolja a meglévő Release Gate-et.
5. Öt `package-test` job jelenik meg: Audacity, VLC Media Player, Krita, Blender, KeePass 2. Mindegyik új `windows-latest` VM-et kap. Az indexeket a Go által generált tervből keresi ki a rendszer.
6. Az `aggregate` job csak öt hitelesített, aktuális buildhez, katalógushoz, futáshoz és VM-hez kötött `FULL_PASS` esetén ír `PILOT_PASS` eredményt. Bármely kihagyás, hiányzó fájl vagy környezeti hiba blokkolja a próbát.

CLI-alternatíva a GitHub-repó hitelesített checkoutjában:

```sh
gh workflow run windows-vm-package-test.yml --ref A_MODOSITASOKAT_TARTALMAZO_AG
gh run list --workflow windows-vm-package-test.yml --limit 5
gh run watch A_KIVALASZTOTT_RUN_ID --exit-status
gh run download A_KIVALASZTOTT_RUN_ID --dir actions-evidence
```

Az ág és a futásazonosító helyére a tényleges érték kerüljön. Ehhez a helyi könyvtárhoz nem volt megadva repó-URL, ezért pontos repository azonosító nem áll rendelkezésre.

## Bizonyítékok

- `pilot-build-RUN-ATTEMPT`: baseline parancsok, kilépési kódok és kimenetek; build manifest; Windows executable; teljes katalógusterv és hash; öt kiválasztott index; üres bizonyítékhalmaz negatív Release Gate riportja.
- `wf-package-RUN-ATTEMPT-INDEX`: az adott VM környezeti bizonyítéka és a program aláírt életciklus-JSON-ja. A teljes JSON-t külön HMAC védi, így a részletes mezők átírása is ellenőrizhető.
- `pilot-report-RUN-ATTEMPT`: összegyűjtött bizonyítékok és `pilot-gate.log`.

A VM-azonosító `github-RUN_ID-RUN_ATTEMPT-INDEX`, a futásazonosító `github-RUN_ID-RUN_ATTEMPT`. A csomagok ugyanazt a buildet kapják; a job induláskor ellenőrzi annak SHA-256 értékét és commitjét. A Go-verziót az `actions/setup-go` a `go.mod` fájlból veszi. Ha a Winget hiányzik, kizárólag az igazolt hosted VM-en a Microsoft dokumentált `Microsoft.WinGet.Client` moduljával történik bootstrap, majd verzió-, forrás- és hálózatellenőrzés.

A `windows-latest` Windows Server képet használ; a tényleges OS, architektúra, képfájlverzió és bootidő a bizonyítékba kerül. Ez az elfogadott Actions-környezeten végzett próba, nem Windows 11 klienskompatibilitási igazolás.

## Megszakítás és újrafuttatás

A program telepítés és eltávolítás előtt atomikusan mentett, aláírt checkpointot készít. Egy megszakított `IN_PROGRESS` állapot soha nem PASS. A hosted VM elvesztése után annak telepített állapota nem állítható vissza a következő jobban.

Folytatáshoz a futás lapján **Re-run all jobs** szükséges: az új attempt friss build-artifactot és öt új VM-et kap. A **Re-run failed jobs** önmagában nem támogatott, mert az új attempt nem használhatja az előző attempt build-/eredmény-artifactjait. Korábbi eredményt nem másolunk át, és nem végzünk puszta eltávolítási folytatást egy másik VM-en.

Rebootot igénylő program nem kap PASS-t. Az Actions workflow nem indít rendszer-újraindítást, és nem állítja, hogy snapshot-visszaállítást végez. A teljes reboot utáni, ugyanazon VM-en végzett folytatás hosted Actions alatt nincs igazolva. A próba öt szokásos desktop csomagra korlátozódik; Store-környezetet igénylő csomagok és védett rendszerkomponensek továbbra is külön, nem sikeres státusszal zárulnak.

Már telepített programot egyik próba sem távolít el. Ha egy új runner-kép előtelepít valamelyik próbaelemet, `SKIPPED_PREEXISTING` és blokkolt próba az eredmény; ilyenkor a próbakészletet külön módosítani kell.

## Elfogadási határ

Az ötprogramos `PILOT_PASS` nem oldja fel a teljes katalógus Release Gate-jét. A teljes kiadáshoz minden szükséges katalóguselemhez aktuális, teljes, hiteles fizikai bizonyíték szükséges. Az aláírás a repository-secret megbízhatóságára támaszkodik; nem véd a titkot vagy az aláíró workflow-t jogosultan módosító féllel szemben.

Helyben csak forrás- és scriptszintű ellenőrzés történt. A Go tesztek, fordítások, negatív kapupróba és öt fizikai életciklus tényleges eredményeit az első Actions-futás fogja szolgáltatni.

Források: [GitHub runner-képek](https://github.com/actions/runner-images), [jobonként új VM](https://docs.github.com/en/actions/reference/runners/github-hosted-runners), [Winget telepítés](https://learn.microsoft.com/en-us/windows/package-manager/winget/).
