param()
Set-ItemProperty -Path "HKLM:\SYSTEM\CurrentControlSet\Services\LDAP" `
  -Name "LdapEnforceChannelBinding" -Value 2
