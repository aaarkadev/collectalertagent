package agents

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/user"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aaarkadev/collectalertagent/internal/configs"
	"github.com/aaarkadev/collectalertagent/internal/repositories"
	"github.com/aaarkadev/collectalertagent/internal/storages"
	"github.com/aaarkadev/collectalertagent/internal/types"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
	"golang.org/x/exp/constraints"
)

func numToFloat[T constraints.Integer | constraints.Float](a T) float64 {
	return float64(a)
}

func updateOne(m types.Metrics, statStructReflect reflect.Value) (types.Metrics, error) {
	switch m.Source {
	case types.IncrementSource:
		{
			err := m.Set((m.GetDelta() + int64(1)))
			if err != nil {
				return m, types.NewTimeError(fmt.Errorf("agent.updateOne(): fail: %w", err))
			}
		}
	case types.RandSource:
		{
			r := rand.New(rand.NewSource(time.Now().UnixNano()))
			err := m.Set(r.Float64())
			if err != nil {
				return m, types.NewTimeError(fmt.Errorf("agent.updateOne(): fail: %w", err))
			}
		}
	default:
		{
			structFieldVal := statStructReflect.FieldByName(m.ID)
			if structFieldVal.IsValid() {
				floatVal := float64(0.0)
				if structFieldVal.CanFloat() {
					floatVal = numToFloat(structFieldVal.Float())
				} else if structFieldVal.CanUint() {
					floatVal = numToFloat(structFieldVal.Uint())
				} else {
					floatVal = numToFloat(structFieldVal.Int())
				}
				err := m.Set(float64(floatVal))
				if err != nil {
					return m, types.NewTimeError(fmt.Errorf("agent.updateOne(): fail: %w", err))
				}
			} else {
				return m, types.NewTimeError(fmt.Errorf("agent.updateOne(): statStructReflect.Value invalid"))
			}
		}
	}

	return m, nil
}

func UpdatePsMetrics(rep repositories.Repo) bool {
	memInfo, _ := mem.VirtualMemory()
	cpuPercents, _ := cpu.Percent(time.Second, true)

	for _, mElem := range rep.GetAll() {
		if !isPsMetrica(mElem) {
			continue
		}

		var err error
		var cpuIdx int
		if mElem.ID == "TotalMemory" {
			err = mElem.Set(float64(memInfo.Total))
		} else if mElem.ID == "FreeMemory" {
			err = mElem.Set(float64(memInfo.Free))
		} else if strings.Index(mElem.ID, "CPUutilization") == 0 {
			cpuIdx, err = strconv.Atoi(strings.SplitN(mElem.ID, "CPUutilization", 2)[1])
			if err != nil {
				continue
			}
			if (cpuIdx-1) < 0 || cpuIdx > len(cpuPercents) {
				continue
			}
			err = mElem.Set(cpuPercents[(cpuIdx - 1)])
		} else {
			continue
		}
		if err != nil {
			log.Println(types.NewTimeError(fmt.Errorf("agent.UpdatePsMetrics(): fail: %w", err)))
		}
		err = rep.Set(mElem)
		if err != nil {
			log.Println(types.NewTimeError(fmt.Errorf("agent.UpdatePsMetrics(): fail: %w", err)))
		}
	}

	return true
}

func startSendMetricsJSON(rep repositories.Repo, config configs.AgentConfig, wg *sync.WaitGroup) {

	var delayedFunc func(isFirstCall bool)
	delayedFunc = func(isFirstCall bool) {
		if !isFirstCall {
			SendMetricsJSON(rep, config)
			isFirstCall = false
		}
		time.AfterFunc(config.ReportInterval, func() {
			delayedFunc(false)
		})
	}

	go func() {
		delayedFunc(true)

		select {
		case <-config.MainCtx.Done():
			wg.Done()
			runtime.Goexit()
			return
		}
	}()

}

func startUpdatePsMetrics(rep repositories.Repo, config configs.AgentConfig, wg *sync.WaitGroup) {

	var delayedFunc func(isFirstCall bool)
	delayedFunc = func(isFirstCall bool) {
		if !isFirstCall {
			UpdatePsMetrics(rep)
		}
		time.AfterFunc(config.PollInterval, func() {
			delayedFunc(false)
		})
	}

	go func() {
		delayedFunc(true)

		select {
		case <-config.MainCtx.Done():
			wg.Done()
			runtime.Goexit()
			return
		}
	}()
}

func startUpdateOsMetrics(rep repositories.Repo, config configs.AgentConfig, wg *sync.WaitGroup) {

	var delayedFunc func(isFirstCall bool)
	delayedFunc = func(isFirstCall bool) {
		if !isFirstCall {
			UpdateOsMetrics(rep)
		}
		time.AfterFunc(config.PollInterval, func() {
			delayedFunc(false)
		})
	}

	go func() {
		delayedFunc(true)

		select {
		case <-config.MainCtx.Done():
			wg.Done()
			runtime.Goexit()
			return
		}
	}()
}

func UpdateOsMetrics(rep repositories.Repo) bool {
	var osStats = runtime.MemStats{}
	runtime.ReadMemStats(&osStats)
	reflectVal := reflect.ValueOf(&osStats)
	if reflectVal.Kind() == reflect.Ptr {
		reflectVal = reflectVal.Elem()
	}
	if reflectVal.Kind() != reflect.Struct {
		return false
	}

	for _, mElem := range rep.GetAll() {
		if isPsMetrica(mElem) {
			continue
		}
		mElem, err := updateOne(mElem, reflectVal)
		if err != nil {
			log.Println(types.NewTimeError(fmt.Errorf("agent.UpdatelMetrics(): fail: %w", err)))
		}
		err = rep.Set(mElem)
		if err != nil {
			log.Println(types.NewTimeError(fmt.Errorf("agent.UpdatelMetrics(): fail: %w", err)))
		}
	}
	return true
}

func isPsMetrica(m types.Metrics) bool {
	if m.ID == "TotalMemory" || m.ID == "FreeMemory" || strings.Index(m.ID, "CPUutilization") == 0 {
		return true
	}
	return false
}

func Init(config *configs.AgentConfig) repositories.Repo {

	rep := storages.MemStorage{}
	rep.Init()
	var initVars = []struct {
		Name   string
		Type   types.DataType
		Source types.DataSource
	}{
		{Name: "Alloc", Type: types.GaugeType, Source: types.OsSource},
		{Name: "BuckHashSys", Type: types.GaugeType, Source: types.OsSource},
		{Name: "Frees", Type: types.GaugeType, Source: types.OsSource},
		{Name: "GCCPUFraction", Type: types.GaugeType, Source: types.OsSource},
		{Name: "GCSys", Type: types.GaugeType, Source: types.OsSource},
		{Name: "HeapAlloc", Type: types.GaugeType, Source: types.OsSource},
		{Name: "HeapIdle", Type: types.GaugeType, Source: types.OsSource},
		{Name: "HeapInuse", Type: types.GaugeType, Source: types.OsSource},
		{Name: "HeapObjects", Type: types.GaugeType, Source: types.OsSource},
		{Name: "HeapReleased", Type: types.GaugeType, Source: types.OsSource},
		{Name: "HeapSys", Type: types.GaugeType, Source: types.OsSource},
		{Name: "LastGC", Type: types.GaugeType, Source: types.OsSource},
		{Name: "Lookups", Type: types.GaugeType, Source: types.OsSource},
		{Name: "MCacheInuse", Type: types.GaugeType, Source: types.OsSource},
		{Name: "MCacheSys", Type: types.GaugeType, Source: types.OsSource},
		{Name: "MSpanInuse", Type: types.GaugeType, Source: types.OsSource},
		{Name: "MSpanSys", Type: types.GaugeType, Source: types.OsSource},
		{Name: "Mallocs", Type: types.GaugeType, Source: types.OsSource},
		{Name: "NextGC", Type: types.GaugeType, Source: types.OsSource},
		{Name: "NumForcedGC", Type: types.GaugeType, Source: types.OsSource},
		{Name: "NumGC", Type: types.GaugeType, Source: types.OsSource},
		{Name: "OtherSys", Type: types.GaugeType, Source: types.OsSource},
		{Name: "PauseTotalNs", Type: types.GaugeType, Source: types.OsSource},
		{Name: "StackInuse", Type: types.GaugeType, Source: types.OsSource},
		{Name: "StackSys", Type: types.GaugeType, Source: types.OsSource},
		{Name: "Sys", Type: types.GaugeType, Source: types.OsSource},
		{Name: "TotalAlloc", Type: types.GaugeType, Source: types.OsSource},
		{Name: "PollCount", Type: types.CounterType, Source: types.IncrementSource},
		{Name: "RandomValue", Type: types.GaugeType, Source: types.RandSource},

		{Name: "TotalMemory", Type: types.GaugeType, Source: types.OsSource},
		{Name: "FreeMemory", Type: types.GaugeType, Source: types.OsSource},
	}

	logicalCnt, _ := cpu.Counts(true)

	for i := 1; i <= logicalCnt; i++ {
		cpuElem := struct {
			Name   string
			Type   types.DataType
			Source types.DataSource
		}{Name: ("CPUutilization" + strconv.Itoa(i)), Type: types.GaugeType, Source: types.OsSource}
		initVars = append(initVars, cpuElem)
	}

	for _, v := range initVars {
		newM, err := types.NewMetric(v.Name, v.Type, v.Source)
		if err != nil {
			log.Fatalln(types.NewTimeError(fmt.Errorf("agent.InitAllMetrics(): fail1: %w", err)))
		}
		err = rep.Set(*newM)
		if err != nil {
			log.Fatalln(types.NewTimeError(fmt.Errorf("agent.InitAllMetrics(): fail2: %w", err)))
		}
	}
	return &rep
}

func SendMetricsJSON(rep repositories.Repo, config configs.AgentConfig) {
	client := &http.Client{}
	client.Timeout = configs.GlobalDefaultTimeout

	sendM := rep.GetAll()
	if len(sendM) < 1 {
		log.Println(types.NewTimeError(fmt.Errorf("agent.SendMetricsJSON(): warn: empty repo")))
		return
	}
	for i := range sendM {
		sendM[i].GenHash(config.HashKey)
	}

	txtM, err := json.Marshal(sendM)

	if err != nil {
		log.Fatalln(types.NewTimeError(fmt.Errorf("agent.SendMetricsJSON(): fail: %w", err)))
	}
	url := fmt.Sprintf("http://%v/updates/", config.SendAddress)
	ctx, cancel := context.WithTimeout(context.Background(), configs.GlobalDefaultTimeout)
	defer cancel()

	req, rqErr := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(txtM))
	if rqErr != nil {
		log.Println(types.NewTimeError(fmt.Errorf("agent.SendMetricsJSON(): warn: %w", rqErr)))
		return
	}
	req.Header.Set("Content-Type", "Content-Type: application/json")

	response, doErr := client.Do(req)
	if doErr != nil {
		log.Println(types.NewTimeError(fmt.Errorf("agent.SendMetricsJSON(): warn: %w", doErr)))
		return
	}

	_, ioErr := io.Copy(io.Discard, response.Body)
	defer response.Body.Close()
	if ioErr != nil {
		log.Println(types.NewTimeError(fmt.Errorf("agent.SendMetricsJSON(): warn: %w", ioErr)))
		return
	}

}

func sendMetricsRaw(rep repositories.Repo, config configs.AgentConfig) {
	client := &http.Client{}
	client.Timeout = configs.GlobalDefaultTimeout

	for _, v := range rep.GetAll() {
		url := fmt.Sprintf("http://%v/update/%v/%v/%v", config.SendAddress, v.MType, v.ID, v.Get())

		ctx, cancel := context.WithTimeout(context.Background(), configs.GlobalDefaultTimeout)
		defer cancel()

		req, rqErr := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
		if rqErr != nil {
			log.Println(types.NewTimeError(fmt.Errorf("agent.sendMetricsRaw(): warn: %w", rqErr)))
			continue
		}
		req.Header.Set("Content-Type", "Content-Type: text/plain")

		response, doErr := client.Do(req)
		if doErr != nil {
			log.Println(types.NewTimeError(fmt.Errorf("agent.sendMetricsRaw(): warn: %w", doErr)))
			continue
		}

		_, ioErr := io.Copy(io.Discard, response.Body)
		if ioErr != nil {
			log.Println(types.NewTimeError(fmt.Errorf("agent.sendMetricsRaw(): warn: %w", ioErr)))
			response.Body.Close()
			continue
		}
		response.Body.Close()
	}
}

var logFile *os.File

func SetupLog() {
	user, err := user.Current()
	if err == nil && user.Username == "dron" {
		logFile, err = os.OpenFile("log.agent.txt", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			panic(fmt.Sprintf("error opening file: %v", err))
		}
	} else {
		logFile = os.Stderr
	}
	log.SetFlags(log.Lshortfile)
	log.SetPrefix("AGENT: ")
	log.SetOutput(logFile)
}

func StopAgent(repo repositories.Repo) {
	repo.Shutdown()
	defer logFile.Close()
}

func StartAgent(rep repositories.Repo, config configs.AgentConfig) {
	var wg sync.WaitGroup

	wg.Add(1)
	startUpdateOsMetrics(rep, config, &wg)
	wg.Add(1)
	startUpdatePsMetrics(rep, config, &wg)
	wg.Add(1)
	startSendMetricsJSON(rep, config, &wg)

	time.AfterFunc(15*time.Second, func() {
		config.MainCtx.Value("mainCtxCancel").(context.CancelFunc)()
	})

	wg.Wait()
}
