package internal

type activityReplay struct {
	history  []activityReplayRecord
	consumed int
}

type activityReplayRecord struct {
	result []any
	err    error
}

func (c *workflowContext) ensureActivityReplay(fnName string) *activityReplay {
	if c.activities == nil {
		c.activities = make(map[string]*activityReplay)
	}
	if entry, ok := c.activities[fnName]; ok {
		return entry
	}
	entry := &activityReplay{}
	c.activities[fnName] = entry
	return entry
}
