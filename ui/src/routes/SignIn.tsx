import { AuthBackdrop, buttonClass, Card, CardBody } from '../ui'
import { API_BASE } from '../api/client'

// "Single GitHub button. No local accounts, no password field, nothing
// else on the page." -- the CPTC design system's own
// sign-in scene, unchanged.
export function SignIn() {
  return (
    <main className="relative grid min-h-dvh place-items-center px-4">
      <AuthBackdrop />
      <div className="relative z-10 w-full max-w-sm">
        <Card className="border-white/60 bg-surface-raised/95 shadow-panel backdrop-blur-xl">
          <CardBody className="flex flex-col items-center gap-6 py-8 text-center">
            <img src="/cptc-icon.png" alt="" width={56} height={56} />
            <h1 className="text-lg font-semibold tracking-tight text-fg">LaForge</h1>
            <a href={`${API_BASE}/auth/github/login`} className={buttonClass({ variant: 'brand', size: 'lg' }) + ' w-full'}>
              Sign In
            </a>
          </CardBody>
        </Card>
      </div>
    </main>
  )
}
