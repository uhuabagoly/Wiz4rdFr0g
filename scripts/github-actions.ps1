param(
 [Parameter(Mandatory=$true)][ValidatePattern('^actions/[A-Za-z0-9_./?=&%,-]+$')][string]$Endpoint,
 [ValidateSet('GET','POST')][string]$Method='GET',
 [string]$BodyFile,
 [string]$OutFile
)
$ErrorActionPreference='Stop'
$env:GIT_TERMINAL_PROMPT='0'
$env:GCM_INTERACTIVE='Never'
# Use the existing GitHub login through the configured credential helper.
# Never print, persist, or transmit this credential outside api.github.com.
$credentialRecord = "protocol=https`nhost=github.com`nusername=uhuabagoly`n`n" | git credential fill
if($LASTEXITCODE -ne 0){throw 'Existing GitHub credential is unavailable'}
$tokenLine=@($credentialRecord | Where-Object {$_ -like 'password=*'})
if($tokenLine.Count -ne 1){throw 'Existing GitHub credential is unavailable'}
$accessToken=$tokenLine[0].Substring(9)
try {
 if($Method -eq 'GET'){
  $separator=if($Endpoint.Contains('?')){'&'}else{'?'}
  $Endpoint+=$separator+'fresh='+[DateTimeOffset]::UtcNow.ToUnixTimeMilliseconds()
 }
 $parameters=@{Uri=('https://api.github.com/repos/uhuabagoly/Wiz4rdFr0g/'+$Endpoint);Method=$Method;Headers=@{Authorization="Bearer $accessToken";Accept='application/vnd.github+json';'X-GitHub-Api-Version'='2022-11-28'};TimeoutSec=120}
 if($BodyFile){$parameters.Body=Get-Content -LiteralPath $BodyFile -Raw;$parameters.ContentType='application/json'}
 if($OutFile){$parameters.OutFile=$OutFile;Invoke-WebRequest @parameters|Out-Null}else{Invoke-RestMethod @parameters|ConvertTo-Json -Depth 30}
} catch {throw ('GitHub Actions request failed: '+$_.Exception.Message)} finally {$accessToken=$null;$credentialRecord=$null;$tokenLine=$null}
