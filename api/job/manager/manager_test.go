package jobmanager

// Public for tests ONLY!
func (jm *JobManager) RunCheckJobQueueForTest() {
	jm.checkJobQueue()
}
