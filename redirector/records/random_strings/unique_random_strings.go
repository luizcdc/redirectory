package unique_random_strings

import (
	"math"
	"math/rand"
	"runtime"
	"sort"
	"sync"

	"github.com/luizcdc/redirectory/redirector/records"
	"github.com/luizcdc/redirectory/redirector/uint_to_any_base"
)

type Generator struct {
	Size         uint32
	numberSystem uint_to_any_base.NumeralSystem
	allAvailable []string
	current      int
}

func NewGenerator(strSize uint32, alphabet []rune) *Generator {
	possibilities := int(math.Pow(float64(len(alphabet)), float64(strSize)))
	numberSystem, err := uint_to_any_base.NewNumeralSystem(uint32(len(alphabet)), string(alphabet), strSize)
	if err != nil {
		return nil
	}
	gen := &Generator{
		Size:         strSize,
		numberSystem: *numberSystem,
		allAvailable: make([]string, possibilities),
		current:      possibilities - 1,
	}
	gen.populate()
	return gen
}

func (gen * Generator) populateRange(start, end int, wg sync.WaitGroup) {
	defer wg.Done()
	currentNumber, _ := gen.numberSystem.IntegerToString(uint32(start))
	for i := start; i < end; i++ {
		gen.allAvailable[i] = currentNumber
		currentNumber, _ = gen.numberSystem.Incr(currentNumber)
	}
}

func (gen *Generator) populate() {
	// To populate from i to i+WINDOW, it should start from gen.numberSystem.IntegerToString(i) and
	// work parallel to the other populateSliceFrom goroutines.
	windowSize := (len(gen.allAvailable) / runtime.NumCPU())+1
	var wg sync.WaitGroup
	for i := 0; i < runtime.NumCPU(); i++ {
		wg.Add(1)
		go gen.populateRange(i*windowSize, max((i+1)*windowSize, len(gen.allAvailable)), wg)
	}
	alreadyUsedCount := 0
	// alreadyUsed falls out of scope as soon as it isn't needed, enabling the GC to reclaim memory
	{
		alreadyUsed := gen.getUsedFromRedis()
		wg.Wait()
		for i, available := range gen.allAvailable {
			_, ok := alreadyUsed[available]
			if ok {
				gen.allAvailable[i] = ""
				alreadyUsedCount++
			}
		}
	}
	sort.Strings(gen.allAvailable)
	oldAllAvailable := gen.allAvailable[alreadyUsedCount:]
	gen.allAvailable = make([]string, len(gen.allAvailable) - alreadyUsedCount)
	copy(gen.allAvailable, oldAllAvailable)
	{
		var tmpString string
		rand.Shuffle(len(gen.allAvailable), func (i,j int) {
			tmpString = gen.allAvailable[i]
			gen.allAvailable[i] = gen.allAvailable[j]
			gen.allAvailable[j] = tmpString
		})
	}
	gen.current = len(gen.allAvailable) - 1
}

func (gen * Generator) getUsedFromRedis() map[string]struct{} {
	// TODO: environment variable for prefix, varying between prod and dev environments
	keys, err := records.GetAllKeysWithoutPrefix("PREFIX")
	if err != nil {
		return map[string]struct{}{}
	}
	allUsed := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		allUsed[key] = struct{}{}
	}
	return allUsed
}

func (gen *Generator) Next() string {
	if gen.current < 0 {
		return ""
	}
	next := gen.allAvailable[gen.current]
	gen.current--
	return next
}