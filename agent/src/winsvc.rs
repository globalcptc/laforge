// Native Windows service support. The agent is a console program; when Windows'
// Service Control Manager launches it (a `sc start`, or boot with `start= auto`),
// it must speak the service protocol -- report RUNNING and handle Stop -- or the
// SCM kills it within ~30s (error 1053). Running under the SCM is what gives
// boot-start and crash-restart (`sc failure` actions) NATIVELY, which a scheduled
// task could only approximate and which a future- or timezone-skewed clock can't
// break (a service has no time trigger).
//
// is_launched_by_scm() connects to the SCM: when the process WAS started as a
// service it runs the service (blocking until stop, so it effectively never
// returns); when it was not (a console/dev run, or the OCI container entrypoint)
// it returns false so main() falls back to the ordinary path. This keeps a single
// binary working as both a service and a plain console program.

use std::ffi::OsString;
use std::sync::OnceLock;
use std::time::Duration;

use windows_service::service::{
    ServiceControl, ServiceControlAccept, ServiceExitCode, ServiceState, ServiceStatus, ServiceType,
};
use windows_service::service_control_handler::{self, ServiceControlHandlerResult, ServiceStatusHandle};
use windows_service::{define_windows_service, service_dispatcher};

// Must match the name the installer registers (see internal/agentdelivery).
const SERVICE_NAME: &str = "LaForgeAgent";

// ERROR_FAILED_SERVICE_CONTROLLER_CONNECT -- the dispatcher returns this when the
// process was NOT started by the SCM (i.e. an ordinary console run), our cue to
// fall back instead of failing.
const ERROR_FAILED_SERVICE_CONTROLLER_CONNECT: i32 = 1063;

// The registered status handle, shared with the control handler so a Stop can
// report STOPPED. Set once, right after registration and before RUNNING, so it is
// always present by the time any control arrives.
static STATUS: OnceLock<ServiceStatusHandle> = OnceLock::new();

define_windows_service!(ffi_service_main, service_main);

// is_launched_by_scm runs as a service when the SCM started us, else returns
// false so the caller runs as a console program. The service path blocks until
// stop (then exits the process), so a `true` return is effectively unreachable.
pub fn is_launched_by_scm() -> bool {
    match service_dispatcher::start(SERVICE_NAME, ffi_service_main) {
        Ok(()) => true,
        Err(windows_service::Error::Winapi(e)) if e.raw_os_error() == Some(ERROR_FAILED_SERVICE_CONTROLLER_CONNECT) => false,
        Err(e) => {
            crate::dlog!("laforge-agent: service dispatcher failed ({e}); running as console");
            false
        }
    }
}

fn service_main(_args: Vec<OsString>) {
    if let Err(e) = run_service() {
        crate::dlog!("laforge-agent: windows service error: {e}");
        std::process::exit(1);
    }
}

fn set_state(state: ServiceState, accept: ServiceControlAccept) {
    if let Some(h) = STATUS.get() {
        let _ = h.set_service_status(ServiceStatus {
            service_type: ServiceType::OWN_PROCESS,
            current_state: state,
            controls_accepted: accept,
            exit_code: ServiceExitCode::Win32(0),
            checkpoint: 0,
            wait_hint: Duration::default(),
            process_id: None,
        });
    }
}

fn run_service() -> Result<(), windows_service::Error> {
    let handler = move |control| match control {
        // The agent loop has no cooperative shutdown, so an operator stop just
        // ends the process. Report STOPPED first, so the SCM treats this as the
        // requested stop rather than a crash (which its failure actions restart).
        ServiceControl::Stop | ServiceControl::Shutdown => {
            set_state(ServiceState::StopPending, ServiceControlAccept::empty());
            set_state(ServiceState::Stopped, ServiceControlAccept::empty());
            std::process::exit(0);
        }
        ServiceControl::Interrogate => ServiceControlHandlerResult::NoError,
        _ => ServiceControlHandlerResult::NotImplemented,
    };
    let handle = service_control_handler::register(SERVICE_NAME, handler)?;
    let _ = STATUS.set(handle);
    set_state(ServiceState::Running, ServiceControlAccept::STOP | ServiceControlAccept::SHUTDOWN);
    // Run the ordinary agent. It blocks forever; if it hits a fatal error and
    // exits non-zero, the SCM's failure actions restart the service -- the native
    // crash recovery a scheduled task can't give.
    crate::run_agent()
}
