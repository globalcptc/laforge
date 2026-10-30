// The status-board palette: fixed hex per status and the dark chrome colors,
// shared by the Matrix (EnvironmentBoard) and the dependency Graph so the two
// read identically on any screen, in any theme. Icons, animations, and labels
// come from StatusBadge; only the solid fill colors live here (the board commits
// to its own palette rather than the app's theme tokens).
export const STATUS_COLOR: Record<string, string> = {
  finished: '#5f941c',
  building: '#7f77dd',
  running: '#3b8ddc',
  deploying: '#2558a8',
  deploy_failed: '#e04a49',
  build_failed: '#d8622f',
  invalid: '#e0a021',
  dependency_blocked: '#e0a021',
  // Composite effective states (an up host with unhealthy agent/infra), folded
  // in by effectiveStatusKey so the board shows a problem, not green.
  agent_missing: '#e04a49',
  agent_late: '#e0a021',
  powered_off: '#d8622f',
  instance_gone: '#e04a49',
  destroy_queued: '#d98fd0',
  destroying: '#d4537e',
  destroy_failed: '#e04a49',
  torn_down: '#5f5e5a',
  destroyed: '#5f5e5a',
}

// Dark chrome tokens for the board surfaces.
export const BOARD = {
  bg: '#0b0714',
  panel: '#17122a',
  panel2: '#120d20',
  line: '#241f33',
  line2: '#2e2947',
  fg: '#e9e6f3',
  muted: '#9089a6',
  faint: '#6c6683',
}

export function statusColor(key: string): string {
  return STATUS_COLOR[key] ?? '#5f5e5a'
}

// A queued/not-yet-created state draws as a dashed ghost rather than a solid
// fill (its matrix tone is near-black ink, invisible on the dark board).
export function isGhostStatus(key: string): boolean {
  return key === 'pending' || key === 'planned'
}

// The legend/column order the board and graph both present statuses in.
export const BOARD_STATUS_ORDER = [
  'pending',
  'deploying',
  'running',
  'building',
  'dependency_blocked',
  'agent_late',
  'agent_missing',
  'powered_off',
  'instance_gone',
  'finished',
  'deploy_failed',
  'build_failed',
  'invalid',
  'destroy_queued',
  'destroying',
  'destroy_failed',
  'torn_down',
]
