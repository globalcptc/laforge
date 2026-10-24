import type { MicrocloudPublicAccess } from '../../api/types'
import { Button, Field, Input } from '../../ui'

export function PublicNICSettings({ value, onChange }: { value: MicrocloudPublicAccess; onChange: (value: MicrocloudPublicAccess) => void }) {
  const update = (patch: Partial<MicrocloudPublicAccess>) => onChange({ ...value, ...patch })
  const routes = value.routes ?? []
  return <div className="flex flex-col gap-4">
    <p className="text-xs text-fg-muted">Each public host gets a second NIC and its own static IPv4 address, with the original port numbers. Linux cloud-init and Windows Cloudbase-Init configure both NICs at first boot. Use an address pool reserved exclusively for LaForge. Existing hosts need rebuilding when switching to this type or changing network settings.</p>
    <div className="grid gap-4 sm:grid-cols-2">
      <Field label="Public OVN network" hint="An existing network, separate from the primary OVN networks." className="mb-0"><Input value={value.network ?? ''} onChange={(e) => update({ network: e.target.value })} placeholder="GUEST_GUAC_WAN" /></Field>
      <Field label="IPv4 subnet" className="mb-0"><Input value={value.cidr ?? ''} onChange={(e) => update({ cidr: e.target.value })} placeholder="10.250.0.0/16" /></Field>
    </div>
    <Field label="Static address pool" hint="Comma-separated addresses or inclusive ranges. Exclude gateways, DNS servers, DHCP pools, and addresses used by other systems." className="mb-0"><Input value={value.ranges ?? ''} onChange={(e) => update({ ranges: e.target.value })} placeholder="10.250.3.100-10.250.3.199" /></Field>
    <div className="grid gap-4 sm:grid-cols-2">
      <Field label="Public default gateway (optional)" hint="Leave empty to keep the default route on the primary NIC." className="mb-0"><Input value={value.gateway ?? ''} onChange={(e) => update({ gateway: e.target.value })} placeholder="10.250.0.1" /></Field>
      <Field label="DNS servers (optional)" hint="Comma-separated IPv4 addresses; empty uses the primary network's DNS." className="mb-0"><Input value={value.dns?.join(',') ?? ''} onChange={(e) => update({ dns: e.target.value.split(',') })} placeholder="10.250.0.1" /></Field>
      <Field label="Guest MTU (optional)" hint="At most the OVN network MTU. Empty inherits the network setting." className="mb-0"><Input type="number" min={576} max={9000} value={value.mtu ?? ''} onChange={(e) => update({ mtu: e.target.value === '' ? undefined : Number(e.target.value) })} placeholder="1500" /></Field>
    </div>
    <div className="flex flex-col gap-2">
      <span className="text-xs font-medium text-fg-muted">Routes through the public NIC (optional)</span>
      {routes.map((route, index) => <div key={index} className="flex items-end gap-2">
        <Field label="Destination CIDR" className="mb-0 flex-1"><Input value={route.to} placeholder="192.0.2.0/24" onChange={(e) => update({ routes: routes.map((r, i) => i === index ? { ...r, to: e.target.value } : r) })} /></Field>
        <Field label="Via gateway" className="mb-0 flex-1"><Input value={route.via} placeholder="10.250.0.1" onChange={(e) => update({ routes: routes.map((r, i) => i === index ? { ...r, via: e.target.value } : r) })} /></Field>
        <Button variant="ghost" onClick={() => update({ routes: routes.filter((_, i) => i !== index) })}>Remove</Button>
      </div>)}
      <div><Button variant="secondary" size="sm" onClick={() => update({ routes: [...routes, { to: '', via: '' }] })}>Add route</Button></div>
    </div>
  </div>
}
