package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/globalcptc/laforge/internal/builder"
	"github.com/globalcptc/laforge/internal/db"
)

// CheckinWatchdog is the never-checked-in watchdog's configuration. A host is
// overdue when its agent has not sent a heartbeat this long after the hoster
// reported the instance created (deployed_object.deployed_at). Windows gets
// its own, longer deadline: sysprep and cloudbase-init routinely take 6-24
// minutes to bring the agent up, against seconds for Linux.
type CheckinWatchdog struct {
	Linux   time.Duration // deadline for a Linux agent; 0 disables the check for Linux
	Windows time.Duration // deadline for a Windows agent; 0 disables the check for Windows
	// Reboot force-reboots an overdue instance through the builder, once. A
	// first boot that wedged (a stuck cloud-init, a hung agent download)
	// usually comes up cleanly on the second try.
	Reboot bool
	// ConsoleBytes caps how much of the instance console log is captured into
	// the event, from the end. 0 skips the capture.
	ConsoleBytes int
}

// DefaultCheckinWatchdog is the shipped configuration: 15 minutes for Linux,
// 30 for Windows, console capture on, no automatic reboot.
var DefaultCheckinWatchdog = CheckinWatchdog{Linux: 15 * time.Minute, Windows: 30 * time.Minute, ConsoleBytes: 16 << 10}

// deadlineFor picks the deadline for an agent platform (an agentdelivery
// target triple such as x86_64-pc-windows-gnu).
func (w CheckinWatchdog) deadlineFor(platform string) time.Duration {
	if strings.Contains(platform, "windows") {
		return w.Windows
	}
	return w.Linux
}

// CheckAgentCheckins flags every host in the build whose agent has not checked
// in by its deadline. Three of 154 hosts in one build sat at `running` with an
// instance and zero heartbeats for over three hours, and nothing noticed;
// their peers' wait loops burned an hour each instead. For each overdue host,
// once per deploy, it:
//
//   - sets deployed_object.agent_checkin_overdue_at, which the API reports as
//     a missing agent (the build's needs-attention list);
//   - captures the tail of the instance console log, when the builder can
//     read one, since the console is the only view of a guest with no agent;
//   - force-reboots the instance, when configured;
//   - writes an agent_checkin_overdue event to the host's timeline with all of
//     the above.
//
// Builder calls are best-effort: a failed console read or reboot is recorded
// on the event, never returned, so one unreachable hoster can't stop the
// watchdog from flagging the rest.
func CheckAgentCheckins(ctx context.Context, pool *pgxpool.Pool, buildID pgtype.UUID, w CheckinWatchdog) error {
	q := db.New(pool)
	candidates, err := q.ListAgentCheckinWatchdogCandidates(ctx, buildID)
	if err != nil {
		return fmt.Errorf("listing check-in watchdog candidates: %w", err)
	}
	now := time.Now()
	var bldr builder.Builder // resolved on the first overdue host that needs it
	var bldrErr error
	resolved := false
	for _, c := range candidates {
		deadline := w.deadlineFor(c.AgentPlatform)
		if deadline <= 0 || !c.DeployedAt.Valid {
			continue
		}
		waited := now.Sub(c.DeployedAt.Time)
		if waited < deadline {
			continue
		}
		// Claim the host before the slow builder calls, so a second
		// orchestrator replica running the same pass doesn't act on it too.
		n, err := q.MarkAgentCheckinOverdue(ctx, c.ID)
		if err != nil {
			return fmt.Errorf("flagging %s as never checked in: %w", c.ID.String(), err)
		}
		if n == 0 {
			continue
		}

		name := db.StrOrEmpty(c.AsName)
		if name == "" {
			name = c.ObjectName
		}
		ref := db.StrOrEmpty(c.ExternalRef)
		team := strconv.FormatInt(int64(c.TeamNumber), 10)
		payload := map[string]any{
			"deployed_at":    c.DeployedAt.Time.UTC(),
			"waited_sec":     int64(waited.Seconds()),
			"deadline_sec":   int64(deadline.Seconds()),
			"agent_platform": c.AgentPlatform,
			"external_ref":   ref,
		}
		var notes []string

		if ref != "" && (w.ConsoleBytes > 0 || w.Reboot) {
			if !resolved {
				bldr, bldrErr = resolveBuilderForBuild(ctx, pool, buildID)
				resolved = true
			}
			if bldrErr != nil {
				payload["builder_error"] = bldrErr.Error()
				notes = append(notes, "builder unavailable")
			} else {
				// Each hoster call bounded: the watchdog runs on the orchestrator's
				// main loop, and a hoster that hangs must not take it along.
				hctx, cancelH := context.WithTimeout(ctx, 30*time.Second)
				if w.ConsoleBytes > 0 {
					if cr, ok := bldr.(builder.ConsoleLogReader); ok {
						if logText, err := cr.ConsoleLog(hctx, team, ref, w.ConsoleBytes); err != nil {
							payload["console_log_error"] = err.Error()
						} else {
							payload["console_log"] = logText
							notes = append(notes, "console log captured")
						}
					}
				}
				if w.Reboot {
					if err := bldr.PowerAction(hctx, team, ref, builder.PowerReboot, true); err != nil {
						payload["reboot_error"] = err.Error()
						notes = append(notes, "forced reboot failed")
					} else {
						payload["rebooted"] = true
						notes = append(notes, "forced a reboot")
					}
				}
				cancelH()
			}
		}

		msg := fmt.Sprintf("agent on %s has not checked in %s after deploy (deadline %s)",
			name, waited.Round(time.Minute), deadline)
		if len(notes) > 0 {
			msg += "; " + strings.Join(notes, ", ")
		}
		body, _ := json.Marshal(payload)
		if _, err := q.CreateObjectEvent(ctx, db.CreateObjectEventParams{
			BuildID: buildID, DeployedObjectID: c.ID, Kind: "agent_checkin_overdue", Message: msg, Payload: body,
		}); err != nil {
			log.Printf("check-in watchdog: writing event for %s: %v", c.ID.String(), err)
		}
		log.Printf("check-in watchdog: build %s: %s", buildID.String(), msg)
	}
	return nil
}
