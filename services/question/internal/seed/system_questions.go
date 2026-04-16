package seed

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	questionservice "easyoffer/question/internal/service"

	"gorm.io/gorm"
)

const (
	SystemQuestionsScriptName = "system_questions_v1"
	systemAuthorID            = "system"
)

type QuestionSeed struct {
	Title        string
	Content      string
	Category     string
	AnswerFormat string
	Language     string
	StarterCode  string
}

func EnsureSystemQuestionsGenerated(db *gorm.DB, svc questionservice.QuestionService) (bool, int, int, string, error) {
	if db == nil {
		return false, 0, 0, "", errors.New("seed database handle is nil")
	}
	if svc == nil {
		return false, 0, 0, "", errors.New("seed question service is nil")
	}

	questions := SystemQuestions()
	total := len(questions)
	hash := questionsScriptHash(questions)

	if err := ensureSeedMetadataTable(db); err != nil {
		return false, 0, total, hash, err
	}

	var existingHash string
	if err := db.Raw(
		"SELECT script_hash FROM question_seed_scripts WHERE script_name = ? LIMIT 1",
		SystemQuestionsScriptName,
	).Scan(&existingHash).Error; err != nil {
		return false, 0, total, hash, fmt.Errorf("failed to load seed script metadata: %w", err)
	}

	if strings.TrimSpace(existingHash) == hash {
		return false, 0, total, hash, nil
	}

	inserted := 0
	for i := range questions {
		q := questions[i]
		_, err := svc.CreateQuestion(
			q.Title,
			q.Content,
			systemAuthorID,
			q.Category,
			q.AnswerFormat,
			q.Language,
			q.StarterCode,
		)
		if err == nil {
			inserted++
			continue
		}
		if errors.Is(err, questionservice.ErrQuestionAlreadyExists) {
			continue
		}

		return false, inserted, total, hash, fmt.Errorf("failed to create seed question %q: %w", q.Title, err)
	}

	now := time.Now().UTC()
	err := db.Exec(
		`INSERT INTO question_seed_scripts (script_name, script_hash, generated_at, total_questions, inserted_questions)
		 VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT (script_name) DO UPDATE
		 SET script_hash = EXCLUDED.script_hash,
		     generated_at = EXCLUDED.generated_at,
		     total_questions = EXCLUDED.total_questions,
		     inserted_questions = EXCLUDED.inserted_questions`,
		SystemQuestionsScriptName,
		hash,
		now,
		total,
		inserted,
	).Error
	if err != nil {
		return false, inserted, total, hash, fmt.Errorf("failed to save seed script metadata: %w", err)
	}

	return true, inserted, total, hash, nil
}

func ensureSeedMetadataTable(db *gorm.DB) error {
	return db.Exec(`CREATE TABLE IF NOT EXISTS question_seed_scripts (
		script_name VARCHAR(128) PRIMARY KEY,
		script_hash VARCHAR(64) NOT NULL,
		generated_at TIMESTAMP NOT NULL,
		total_questions INTEGER NOT NULL,
		inserted_questions INTEGER NOT NULL
	)`).Error
}

func SystemQuestions() []QuestionSeed {
	questions := make([]QuestionSeed, 0, 150)
	questions = append(questions, baseQuestions()...)
	questions = append(questions, generatedTextQuestions()...)
	questions = append(questions, generatedCodeQuestions()...)
	return questions
}

func questionsScriptHash(questions []QuestionSeed) string {
	var b strings.Builder
	for _, q := range questions {
		b.WriteString(strings.TrimSpace(q.Title))
		b.WriteString("|")
		b.WriteString(strings.TrimSpace(q.Content))
		b.WriteString("|")
		b.WriteString(strings.TrimSpace(q.Category))
		b.WriteString("|")
		b.WriteString(strings.TrimSpace(q.AnswerFormat))
		b.WriteString("|")
		b.WriteString(strings.TrimSpace(q.Language))
		b.WriteString("|")
		b.WriteString(strings.TrimSpace(q.StarterCode))
		b.WriteString("\n")
	}

	sum := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(sum[:])
}

func baseQuestions() []QuestionSeed {
	return []QuestionSeed{
		{
			Title:        "Что такое goroutine и чем она отличается от потока ОС?",
			Content:      "Объясните что такое goroutine в Go. Как они реализованы внутри runtime? Чем goroutine отличается от потоков операционной системы?",
			Category:     "golang",
			AnswerFormat: "text",
			Language:     "go",
		},
		{
			Title:        "Как работает scheduler в Go?",
			Content:      "Опишите модель планирования goroutine M:N. Что означают M, P и G в планировщике Go?",
			Category:     "golang",
			AnswerFormat: "text",
			Language:     "go",
		},
		{
			Title:        "Slice vs Array",
			Content:      "Чем slice отличается от массива в Go? Что хранится внутри slice?",
			Category:     "golang",
			AnswerFormat: "text",
			Language:     "go",
		},
		{
			Title:        "Как работает append?",
			Content:      "Опишите что происходит внутри функции append. Когда происходит realloc backing array?",
			Category:     "golang",
			AnswerFormat: "text",
			Language:     "go",
		},
		{
			Title:        "Что такое interface в Go?",
			Content:      "Объясните как реализованы интерфейсы в Go. Что происходит при передаче интерфейса в функцию?",
			Category:     "golang",
			AnswerFormat: "text",
			Language:     "go",
		},
		{
			Title:        "Buffered vs Unbuffered channels",
			Content:      "Чем отличаются buffered и unbuffered каналы? Когда использовать каждый из них?",
			Category:     "concurrency",
			AnswerFormat: "text",
			Language:     "go",
		},
		{
			Title:        "Deadlock в Go",
			Content:      "Что такое deadlock? Приведите пример deadlock в Go и объясните как его избежать.",
			Category:     "concurrency",
			AnswerFormat: "text",
			Language:     "go",
		},
		{
			Title:        "Реализовать worker pool",
			Content:      "Реализуйте worker pool на Go, который обрабатывает задачи из канала.",
			Category:     "algorithms",
			AnswerFormat: "code",
			Language:     "go",
			StarterCode: `func WorkerPool(jobs []int, workers int) []int {
	// implement
	return nil
}`,
		},
		{
			Title:        "Реализовать reverse строки",
			Content:      "Напишите функцию, которая разворачивает строку.",
			Category:     "algorithms",
			AnswerFormat: "code",
			Language:     "go",
			StarterCode: `func ReverseString(s string) string {
	// implement
	return ""
}`,
		},
		{
			Title:        "Two Sum",
			Content:      "Дан массив чисел и target. Нужно вернуть индексы двух чисел, сумма которых равна target.",
			Category:     "algorithms",
			AnswerFormat: "code",
			Language:     "go",
			StarterCode: `func TwoSum(nums []int, target int) []int {
	// implement
	return nil
}`,
		},
		{
			Title:        "Что такое idempotency?",
			Content:      "Объясните что такое идемпотентность. Почему она важна в распределённых системах?",
			Category:     "system_design",
			AnswerFormat: "text",
			Language:     "go",
		},
		{
			Title:        "Outbox pattern",
			Content:      "Объясните Outbox pattern. Как он решает проблему потери событий между БД и Kafka?",
			Category:     "system_design",
			AnswerFormat: "text",
			Language:     "go",
		},
		{
			Title:        "Что такое consumer group в Kafka?",
			Content:      "Как работает consumer group в Kafka? Как распределяются партиции между консьюмерами?",
			Category:     "kafka",
			AnswerFormat: "text",
			Language:     "go",
		},
		{
			Title:        "Что такое consumer lag?",
			Content:      "Что такое consumer lag и как его мониторить?",
			Category:     "kafka",
			AnswerFormat: "text",
			Language:     "go",
		},
		{
			Title:        "Redis cache aside pattern",
			Content:      "Опишите cache-aside pattern. Как он работает с Redis?",
			Category:     "redis",
			AnswerFormat: "text",
			Language:     "go",
		},
		{
			Title:        "LRU Cache",
			Content:      "Реализуйте LRU cache.",
			Category:     "algorithms",
			AnswerFormat: "code",
			Language:     "go",
			StarterCode: `type LRUCache struct {
}

func (c *LRUCache) Get(key int) int {
	return -1
}

func (c *LRUCache) Put(key int, value int) {

}`,
		},
		{
			Title:        "Что такое race condition?",
			Content:      "Объясните race condition. Как обнаружить race condition в Go?",
			Category:     "concurrency",
			AnswerFormat: "text",
			Language:     "go",
		},
		{
			Title:        "sync.Mutex vs sync.RWMutex",
			Content:      "Чем отличается Mutex от RWMutex? Когда лучше использовать RWMutex?",
			Category:     "concurrency",
			AnswerFormat: "text",
			Language:     "go",
		},
		{
			Title:        "Binary Search",
			Content:      "Реализуйте бинарный поиск.",
			Category:     "algorithms",
			AnswerFormat: "code",
			Language:     "go",
			StarterCode: `func BinarySearch(nums []int, target int) int {
	// implement
	return -1
}`,
		},
		{
			Title:        "Что такое CQRS?",
			Content:      "Объясните паттерн CQRS. Когда его стоит использовать?",
			Category:     "system_design",
			AnswerFormat: "text",
			Language:     "go",
		},
	}
}

func generatedTextQuestions() []QuestionSeed {
	type style struct {
		titleSuffix string
		contentFmt  string
	}

	styles := []style{
		{
			titleSuffix: "теория и устройство",
			contentFmt:  "Раскройте тему \"%s\" в категории %s: базовые понятия, внутреннее устройство и ключевые термины.",
		},
		{
			titleSuffix: "production-практика",
			contentFmt:  "Как применять \"%s\" в production в категории %s: архитектурные решения, мониторинг, тестирование, rollback-план?",
		},
		{
			titleSuffix: "ошибки и диагностика",
			contentFmt:  "Какие типичные ошибки возникают в теме \"%s\" (категория %s) и как их диагностировать по метрикам/логам/трейсам?",
		},
	}

	subjectsByCategory := map[string][]string{
		"golang": {
			"Goroutine lifecycle",
			"Go scheduler M:P:G",
			"Garbage collector phases",
			"Escape analysis",
			"Interface internals",
			"Generics constraints",
			"Context propagation",
			"Panic and recover",
		},
		"concurrency": {
			"Channel patterns",
			"WaitGroup orchestration",
			"Mutex contention",
			"Atomic primitives",
			"Race detector usage",
			"Cancellation patterns",
			"Backpressure strategy",
		},
		"algorithms": {
			"Binary search variants",
			"Hash map optimization",
			"Sliding window",
			"Two pointers technique",
			"Heap usage",
			"BFS vs DFS",
			"Dynamic programming decomposition",
			"Prefix sums",
		},
		"system_design": {
			"CQRS boundaries",
			"Event sourcing replay",
			"Outbox consistency",
			"Saga orchestration",
			"Idempotency keys",
			"Rate limiting strategy",
			"Service decomposition",
		},
		"kafka": {
			"Consumer group balancing",
			"Partition strategy",
			"Offset management",
			"Rebalance tuning",
			"Retention and compaction",
		},
		"redis": {
			"Cache-aside lifecycle",
			"Eviction policies",
			"Lua scripts in Redis",
			"Distributed locks",
			"Redis Streams",
		},
	}

	categoryOrder := []string{"golang", "concurrency", "algorithms", "system_design", "kafka", "redis"}

	questions := make([]QuestionSeed, 0, 120)
	for _, category := range categoryOrder {
		subjects := subjectsByCategory[category]
		for _, subject := range subjects {
			for _, st := range styles {
				questions = append(questions, QuestionSeed{
					Title:        fmt.Sprintf("%s: %s", subject, st.titleSuffix),
					Content:      fmt.Sprintf(st.contentFmt, subject, category),
					Category:     category,
					AnswerFormat: "text",
					Language:     "go",
				})
			}
		}
	}

	return questions
}

func generatedCodeQuestions() []QuestionSeed {
	return []QuestionSeed{
		{
			Title:        "Реализовать Min Stack",
			Content:      "Реализуйте структуру MinStack с операциями Push, Pop, Top и GetMin за O(1).",
			Category:     "algorithms",
			AnswerFormat: "code",
			Language:     "go",
			StarterCode: `type MinStack struct {
}

func (s *MinStack) Push(x int) {
}

func (s *MinStack) Pop() {
}

func (s *MinStack) Top() int {
	return 0
}

func (s *MinStack) GetMin() int {
	return 0
}`,
		},
		{
			Title:        "Top K Frequent Elements",
			Content:      "Найдите k самых частых элементов массива.",
			Category:     "algorithms",
			AnswerFormat: "code",
			Language:     "go",
			StarterCode: `func TopKFrequent(nums []int, k int) []int {
	// implement
	return nil
}`,
		},
		{
			Title:        "Token Bucket Rate Limiter",
			Content:      "Реализуйте потокобезопасный token bucket limiter на Go.",
			Category:     "system_design",
			AnswerFormat: "code",
			Language:     "go",
			StarterCode: `type TokenBucket struct {
}

func NewTokenBucket(capacity int, refillPerSecond int) *TokenBucket {
	return &TokenBucket{}
}

func (b *TokenBucket) Allow() bool {
	return false
}`,
		},
		{
			Title:        "Fan-In для нескольких каналов",
			Content:      "Объедините несколько входных каналов в один выходной канал.",
			Category:     "concurrency",
			AnswerFormat: "code",
			Language:     "go",
			StarterCode: `func FanIn(channels ...<-chan int) <-chan int {
	out := make(chan int)
	// implement
	return out
}`,
		},
		{
			Title:        "Context-aware worker",
			Content:      "Сделайте worker, который завершает обработку по отмене context.",
			Category:     "concurrency",
			AnswerFormat: "code",
			Language:     "go",
			StarterCode: `func ProcessWithContext(ctx context.Context, jobs <-chan int, handle func(int) error) error {
	// implement
	return nil
}`,
		},
		{
			Title:        "Retry с exponential backoff",
			Content:      "Реализуйте функцию retry с max attempts и exponential backoff.",
			Category:     "system_design",
			AnswerFormat: "code",
			Language:     "go",
			StarterCode: `func Retry(maxAttempts int, fn func() error) error {
	// implement
	return nil
}`,
		},
		{
			Title:        "Очередь на двух стеках",
			Content:      "Реализуйте Queue используя два стека.",
			Category:     "algorithms",
			AnswerFormat: "code",
			Language:     "go",
			StarterCode: `type Queue struct {
}

func (q *Queue) Push(x int) {
}

func (q *Queue) Pop() int {
	return 0
}

func (q *Queue) Empty() bool {
	return true
}`,
		},
		{
			Title:        "Leaky Bucket Limiter",
			Content:      "Реализуйте leaky bucket limiter с фиксированной скоростью утечки.",
			Category:     "system_design",
			AnswerFormat: "code",
			Language:     "go",
			StarterCode: `type LeakyBucket struct {
}

func NewLeakyBucket(capacity int, leakPerSecond int) *LeakyBucket {
	return &LeakyBucket{}
}

func (b *LeakyBucket) Add(tokens int) bool {
	return false
}`,
		},
		{
			Title:        "Дедупликация событий по event_id",
			Content:      "Реализуйте in-memory дедупликацию событий с TTL по event_id.",
			Category:     "kafka",
			AnswerFormat: "code",
			Language:     "go",
			StarterCode: `type EventDeduper struct {
}

func NewEventDeduper(ttl time.Duration) *EventDeduper {
	return &EventDeduper{}
}

func (d *EventDeduper) Seen(eventID string) bool {
	return false
}`,
		},
		{
			Title:        "In-memory LRU с mutex",
			Content:      "Реализуйте thread-safe LRU cache на map + doubly linked list.",
			Category:     "redis",
			AnswerFormat: "code",
			Language:     "go",
			StarterCode: `type LRU struct {
}

func NewLRU(capacity int) *LRU {
	return &LRU{}
}

func (l *LRU) Get(key string) (string, bool) {
	return "", false
}

func (l *LRU) Put(key, value string) {
}`,
		},
	}
}
