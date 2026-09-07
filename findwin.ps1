Add-Type -TypeDefinition @"
using System;
using System.Text;
using System.Runtime.InteropServices;
public class WinEnum {
  public delegate bool Cb(IntPtr h, IntPtr l);
  [DllImport("user32.dll")] public static extern bool EnumWindows(Cb cb, IntPtr l);
  [DllImport("user32.dll")] public static extern int GetWindowTextW(IntPtr h, StringBuilder s, int n);
  [DllImport("user32.dll")] public static extern bool IsWindowVisible(IntPtr h);
}
"@
$cb = { param($h, $l)
  if ([WinEnum]::IsWindowVisible($h)) {
    $sb = New-Object -TypeName Text.StringBuilder -ArgumentList 256
    [WinEnum]::GetWindowTextW($h, $sb, 256) | Out-Null
    $t = $sb.ToString()
    if ($t -like "*Berwin*") { Write-Output "FOUND-VISIBLE-WINDOW: $t" }
  }
  return $true
}
[WinEnum]::EnumWindows($cb, [IntPtr]::Zero) | Out-Null
