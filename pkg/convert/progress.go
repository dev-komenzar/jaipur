package converter

import (
	"fmt"
	"sync"
	"sync/atomic"
)

// ProgressCounter はスレッドセーフな進捗カウンターを提供
type ProgressCounter struct {
	current int64
	total   int64
	mu      sync.Mutex
}

// NewProgressCounter は新しい ProgressCounter を作成
func NewProgressCounter(total int) *ProgressCounter {
	return &ProgressCounter{total: int64(total)}
}

// Increment は進捗を1つ進めて表示を更新
func (p *ProgressCounter) Increment() {
	current := atomic.AddInt64(&p.current, 1)
	p.mu.Lock()
	defer p.mu.Unlock()
	fmt.Printf("\r%d/%d", current, p.total)
}

// Done は最終的な改行を出力
func (p *ProgressCounter) Done() {
	p.mu.Lock()
	defer p.mu.Unlock()
	fmt.Println()
}
