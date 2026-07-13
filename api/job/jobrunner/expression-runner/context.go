package expressionrunner

import "sync"

// We need a way for Go functions called from lua to find the state they're calling into so we know what
// quant/scan etc to load. To do this simply, we define a global map of context id to our own state
// and clear it when the lua VM is cleaned up

var expressionContexts = map[int]*expressionRunner{}
var expressionContextNextId = 1
var expressionContextMutex sync.Mutex

func addExpressionContext(e *expressionRunner) int {
	expressionContextMutex.Lock()
	defer expressionContextMutex.Unlock()

	id := expressionContextNextId
	expressionContextNextId++

	expressionContexts[id] = e
	return id
}

func clearExpressionContext(id int) {
	expressionContextMutex.Lock()
	defer expressionContextMutex.Unlock()
	delete(expressionContexts, id)
}

func getExpressionContext(id int) *expressionRunner {
	expressionContextMutex.Lock()
	defer expressionContextMutex.Unlock()

	return expressionContexts[id]
}
