// Basic host metrics sampled on each heartbeat, collected uniformly across
// Linux and Windows via sysinfo: CPU %, memory %, disk usage %, and network
// throughput (bytes/sec since the previous sample). Inside a container the
// agent is PID 1, so sysinfo reads the container's own namespaced view -- these
// are per-object metrics (the container's or the host's), exactly what the
// heartbeats panel wants.
//
// A single long-lived Collector per agent session: CPU and network are deltas
// since the last refresh, so keeping the sysinfo handles between heartbeats is
// what makes the numbers an average over each ~heartbeat interval rather than a
// cold first read. Reads are best-effort -- a facet that can't be read just
// leaves its field None, never fails the heartbeat.

use crate::protocol::HeartbeatRequestPayload;
use std::time::Instant;
use sysinfo::{Disks, Networks, System};

pub struct Collector {
    sys: System,
    nets: Networks,
    last: Instant,
}

impl Collector {
    pub fn new() -> Self {
        let mut sys = System::new();
        sys.refresh_cpu_usage();
        sys.refresh_memory();
        Collector {
            sys,
            nets: Networks::new_with_refreshed_list(),
            last: Instant::now(),
        }
    }

    pub fn sample(&mut self) -> HeartbeatRequestPayload {
        let now = Instant::now();
        let elapsed = now.duration_since(self.last).as_secs_f64().max(0.001);
        self.last = now;

        self.sys.refresh_cpu_usage();
        self.sys.refresh_memory();
        let cpu = self.sys.global_cpu_usage() as f64;
        let total_mem = self.sys.total_memory();
        let mem = if total_mem > 0 {
            self.sys.used_memory() as f64 / total_mem as f64 * 100.0
        } else {
            0.0
        };

        // Disk usage %, aggregated across all mounted filesystems -- a single
        // consistent "how full is this host" number on either OS.
        let disks = Disks::new_with_refreshed_list();
        let (mut dtotal, mut davail): (u64, u64) = (0, 0);
        for d in &disks {
            dtotal += d.total_space();
            davail += d.available_space();
        }
        let disk = if dtotal > 0 {
            (dtotal.saturating_sub(davail)) as f64 / dtotal as f64 * 100.0
        } else {
            0.0
        };

        // Network throughput: received()/transmitted() are bytes since the last
        // refresh of this persistent Networks, so dividing by the elapsed time
        // gives a per-second rate over the heartbeat interval.
        self.nets.refresh(true);
        let (mut rx, mut tx): (u64, u64) = (0, 0);
        for (_name, data) in &self.nets {
            rx += data.received();
            tx += data.transmitted();
        }

        HeartbeatRequestPayload {
            cpu_pct: Some(cpu),
            mem_pct: Some(mem),
            disk_pct: Some(disk),
            net_rx_bps: Some(rx as f64 / elapsed),
            net_tx_bps: Some(tx as f64 / elapsed),
        }
    }
}
