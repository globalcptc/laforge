// A bounded, non-blocking buffer of the supervised app's console lines. The
// reader threads (one per stdout/stderr) push here; the gateway loop drains it
// and ships batches as LogBatchRequest frames. Bounded on purpose: a chatty or
// runaway app must never block (pushing is O(1) and drops the oldest line when
// full) nor grow memory without limit. Dropped lines are counted, not hidden --
// the next drained batch leads with a synthetic record saying how many were
// lost, so a flood shows up as a visible gap instead of silence.

use crate::protocol::LogLine;
use std::collections::VecDeque;
use std::sync::Mutex;
use std::time::{SystemTime, UNIX_EPOCH};

pub struct LogQueue {
    inner: Mutex<Inner>,
    cap: usize,
}

struct Inner {
    lines: VecDeque<LogLine>,
    dropped: usize,
}

impl LogQueue {
    pub fn new(cap: usize) -> Self {
        LogQueue {
            inner: Mutex::new(Inner { lines: VecDeque::new(), dropped: 0 }),
            cap,
        }
    }

    /// Push one captured line. Non-blocking; drops the oldest line (counted)
    /// when the buffer is at capacity. A poisoned lock is ignored -- losing a
    /// log line must never take the agent down.
    pub fn push(&self, stream: &'static str, line: String) {
        let mut inner = match self.inner.lock() {
            Ok(g) => g,
            Err(_) => return,
        };
        if inner.lines.len() >= self.cap {
            inner.lines.pop_front();
            inner.dropped += 1;
        }
        let ts_ms = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .map(|d| d.as_millis() as i64)
            .unwrap_or(0);
        inner.lines.push_back(LogLine { ts_ms, stream, line, dropped: 0 });
    }

    /// Take up to `max` buffered lines. If any were dropped since the last
    /// drain, the returned batch leads with a synthetic marker record carrying
    /// that count, then the drop counter resets. Empty when nothing is buffered
    /// and nothing was dropped.
    pub fn drain_batch(&self, max: usize) -> Vec<LogLine> {
        let mut inner = match self.inner.lock() {
            Ok(g) => g,
            Err(_) => return Vec::new(),
        };
        let dropped = std::mem::take(&mut inner.dropped);
        let mut out = Vec::new();
        if dropped > 0 {
            let ts_ms = SystemTime::now()
                .duration_since(UNIX_EPOCH)
                .map(|d| d.as_millis() as i64)
                .unwrap_or(0);
            out.push(LogLine { ts_ms, stream: "stderr", line: String::new(), dropped });
        }
        let take = max.min(inner.lines.len());
        out.extend(inner.lines.drain(..take));
        out
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn drops_oldest_when_full_and_reports_count() {
        let q = LogQueue::new(2);
        q.push("stdout", "a".into());
        q.push("stdout", "b".into());
        q.push("stdout", "c".into()); // evicts "a"
        let batch = q.drain_batch(10);
        // Leading marker for the 1 dropped, then "b", "c".
        assert_eq!(batch.len(), 3);
        assert_eq!(batch[0].dropped, 1);
        assert_eq!(batch[1].line, "b");
        assert_eq!(batch[2].line, "c");
        // Drop counter reset; a fresh drain has no marker.
        q.push("stdout", "d".into());
        let batch = q.drain_batch(10);
        assert_eq!(batch.len(), 1);
        assert_eq!(batch[0].line, "d");
        assert_eq!(batch[0].dropped, 0);
    }

    #[test]
    fn empty_drain_is_empty() {
        let q = LogQueue::new(4);
        assert!(q.drain_batch(10).is_empty());
    }

    #[test]
    fn respects_batch_max() {
        let q = LogQueue::new(100);
        for i in 0..10 {
            q.push("stdout", format!("line {i}"));
        }
        let batch = q.drain_batch(4);
        assert_eq!(batch.len(), 4);
        assert_eq!(batch[0].line, "line 0");
        // The rest remain for the next drain.
        assert_eq!(q.drain_batch(100).len(), 6);
    }
}
