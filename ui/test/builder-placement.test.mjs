import { test } from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import ts from 'typescript'

// Exercise the wizard model without adding a browser runner or requiring Node's
// native TypeScript support (the existing development dependency compiles it).
const source = readFileSync(new URL('../src/components/builder-wizard/model.ts', import.meta.url), 'utf8')
const compiled = ts.transpileModule(source, {compilerOptions: {module: ts.ModuleKind.ESNext}}).outputText
const { placementDefaults, isHostPlaced, uplinkCandidates, draftFromConfig, toRequest } = await import('data:text/javascript;base64,' + Buffer.from(compiled).toString('base64'))

const connection = (supported) => ({ discovery: {
  storage_pools: [{name: 'default'}], native_bridge_supported: supported,
  networks: [{name: 'incusbr0', type: 'bridge', managed: true}, {name: 'eth0', type: 'physical', managed: false}, {name: 'UPLINK', type: 'physical', managed: true}, {name:'ovn0',type:'ovn',managed:true}],
} })

test('Incus detects bridges; MicroCloud keeps an OVN uplink', () => {
  const host = placementDefaults(connection(true), 'incus')
  assert.equal(host.networkType, 'bridge')
  assert.ok(isHostPlaced({...host, uplink:'', connection:connection(true)}))
  assert.equal(placementDefaults(connection(false), 'incus').networkType, 'ovn')
  assert.equal(isHostPlaced({...host, connection:connection(false)}), false)
  assert.equal(placementDefaults(connection(true), 'microcloud').networkType, undefined)
  assert.deepEqual(uplinkCandidates(connection(true).discovery.networks).map(n=>n.name), ['incusbr0','UPLINK'])
})

test('saved OVN settings stay OVN; bridge and strict placement round-trip', () => {
  const config = {name:'test',kind:'incus',incus_images:{},incus_sizes:{},incus_hosts:[{credential_id:'host-id',storage_pool:'default',ovn_uplink_network:'UPLINK'}]}
  const legacy = draftFromConfig(config)
  assert.equal(legacy.hosts[0].networkType,'ovn')
  assert.equal(toRequest(legacy).incus_hosts[0].ovn_uplink_network,'UPLINK')
  const draft = draftFromConfig({...config,incus_one_team_per_host:true,incus_hosts:[{...config.incus_hosts[0],network_type:'bridge'}]})
  const saved = toRequest(draft)
  assert.equal(saved.incus_one_team_per_host,true)
  assert.equal(saved.incus_hosts[0].network_type,'bridge')
  assert.equal(saved.incus_hosts[0].ovn_uplink_network,undefined)
})
