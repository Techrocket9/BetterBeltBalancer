package main

const maxCompileRetries = 3

type compileResult uint8

const (
	compileDone compileResult = iota
	compileRefused
	compileRetry
)

// Membership is attached to the predecessor node, while pending is its sum at
// the current component root. Union and split maintain it without global scans.
type queuedWork struct {
	dead, stranded, live   bool
	retired, releaseQueued bool
	pending                uint32
	strandedN              uint32
	retries                uint8
}

var workQueue []queuedWork
var retryRoots []uint32
var retiredCandidates []uint32

// Only transitions that can release an identity enqueue it. A long-lived
// stranded predecessor is not polled by unrelated edits or ordinary flushes.
func queueRetiredNode(r uint32) {
	w := &workQueue[r]
	if !w.retired || w.releaseQueued {
		return
	}
	w.releaseQueued = true
	retiredCandidates = append(retiredCandidates, r)
}

func releaseRetiredNodes() {
	for _, r := range retiredCandidates {
		w := &workQueue[r]
		w.releaseQueued = false
		if alive[r] || !w.retired || w.dead || w.stranded || w.live {
			continue
		}
		if _, owned := nets[r]; owned {
			continue
		}
		w.retired = false
		freeID = append(freeID, r)
	}
	retiredCandidates = retiredCandidates[:0]
}

func setPending(r uint32, dead, on bool) {
	w := &workQueue[r]
	flag := &w.stranded
	if dead {
		flag = &w.dead
	}
	if *flag == on {
		return
	}
	*flag = on
	if !on {
		queueRetiredNode(r)
	}
	if alive[r] {
		root := find(r)
		if on {
			workQueue[root].pending++
		} else {
			workQueue[root].pending--
		}
		if !dead {
			if on {
				workQueue[root].strandedN++
			} else {
				workQueue[root].strandedN--
			}
		}
	}
}

func ownPending(r uint32) uint32 {
	n := uint32(0)
	if workQueue[r].dead {
		n++
	}
	if workQueue[r].stranded {
		n++
	}
	return n
}

// Lifecycle must use this instead of directly truncating the two queues.
// Stranded ownership is independent and is deliberately retained.
func resetWorkQueues() {
	for _, r := range deadRoots {
		setPending(r, true, false)
	}
	for _, r := range liveRoots {
		workQueue[r].live = false
		queueRetiredNode(r)
	}
	deadRoots, liveRoots = deadRoots[:0], liveRoots[:0]
	retryRoots = retryRoots[:0]
	preflightBlocked = preflightBlocked[:0]
}

// Internal scheduling preserves the retry budget. queueLive is the external
// edit entry point and starts a new bounded attempt sequence.
func enqueueLive(r uint32) {
	if workQueue[r].live {
		return
	}
	workQueue[r].live = true
	liveRoots = append(liveRoots, r)
}

func finishCompile(r uint32, result compileResult) {
	w := &workQueue[r]
	noteBuildResult(r, result, w.retries >= maxCompileRetries)
	if result != compileRetry {
		w.retries = 0
		return
	}
	if w.retries < maxCompileRetries {
		w.retries++
		retryRoots = append(retryRoots, r)
		return
	}
	if w.retries == maxCompileRetries {
		w.retries++
		logAlertStart("cluster ")
		logU(r)
		logS(" compile retries exhausted; retaining classification work until the next edit or audit")
		logEnd()
	}
}

func publishCompileRetries() {
	for _, r := range retryRoots {
		if alive[r] {
			enqueueLive(find(r))
		}
	}
	if len(retryRoots) != 0 {
		requestFlush()
	}
	retryRoots = retryRoots[:0]
}
