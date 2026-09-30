package releaseproof

// LICEcapIndependentScript is the exact independent observation command accepted by the gate.
const LICEcapIndependentScript = `$ErrorActionPreference='Stop'
try {
  $pf=[Environment]::GetEnvironmentVariable('ProgramFiles(x86)')
  if ([string]::IsNullOrWhiteSpace($pf) -or -not [IO.Path]::IsPathRooted($pf)) { throw 'Missing machine program directory' }
  $root=[IO.Path]::Combine($pf,'LICEcap')
  $base=[Microsoft.Win32.RegistryKey]::OpenBaseKey([Microsoft.Win32.RegistryHive]::LocalMachine,[Microsoft.Win32.RegistryView]::Registry32)
  try {
    $key=$base.OpenSubKey('Software\LICEcap')
    $registered=$null
    if ($null -ne $key) { try { $registered=$key.GetValue('') } finally { $key.Dispose() } }
  } finally { $base.Dispose() }
  $files=@()
  foreach ($name in @('LICEcap.exe','Uninstall.exe')) {
    $path=[IO.Path]::Combine($root,$name)
    $stream=$null
    try { $stream=[IO.File]::Open($path,[IO.FileMode]::Open,[IO.FileAccess]::Read,[IO.FileShare]::ReadWrite) }
    catch [IO.FileNotFoundException] { continue }
    catch [IO.DirectoryNotFoundException] { continue }
    try {
      $reader=[IO.BinaryReader]::new($stream)
      $valid=$false
      if ($stream.Length -ge 64 -and $reader.ReadUInt16() -eq 0x5A4D) {
        $stream.Position=60; $offset=$reader.ReadUInt32()
        if ($offset -ge 64 -and $offset -le ($stream.Length-4)) { $stream.Position=$offset; $valid=($reader.ReadUInt32() -eq 0x4550) }
      }
      $files+=@{path=$path; bytes=$stream.Length; pe=$valid}
    } finally { $stream.Dispose() }
  }
  $state='unknown'
  if ($null -eq $registered -and $files.Count -eq 0) { $state='absent' }
  elseif ($registered -is [string] -and [IO.Path]::GetFullPath($registered).TrimEnd('\') -ieq $root -and $files.Count -eq 2 -and @($files | Where-Object { -not $_.pe }).Count -eq 0) { $state='present' }
  @{state=$state; registry='HKEY_LOCAL_MACHINE\Software\LICEcap'; view=32; registered_root=$registered; files=$files} | ConvertTo-Json -Depth 4 -Compress
} catch { Write-Error $_; exit 1 }`
