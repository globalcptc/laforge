package scheduler

import (
	"fmt"

	"github.com/gen0cide/laforge/ent"
)

func FromStepToProvisinedStep(client *ent.Client, step *ent.ScheduleStep) {
	for {
		// Add check for status

		// Check if repeated step
		if step.Repeated {
			delta_time := step.EndTime.Sub(step.StartTime).Nanoseconds()
			interval := step.Interval
			total := delta_time / int64(interval)
			fmt.Println(delta_time)
			fmt.Println(interval)
			fmt.Println(total)

			// client.ProvisinedScheduleStep.Create().SetType().StepNumber()
		} else {

		}
	}
}

func main() {

}
