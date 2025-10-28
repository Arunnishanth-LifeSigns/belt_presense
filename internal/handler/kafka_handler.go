package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"belt-presense/internal/database"
	"belt-presense/internal/models"
)

const chunkSize = 30

type PatientBatch struct {
	Messages []*models.ECGMessage
}

type CachedVitals struct {
	BP          models.BloodPressure
	SPO2        models.VitalSign
	PR          models.VitalSign
	DeviceID    string
	LastUpdated int64
}

type BeltProcessor struct {
	db                  *database.Repository
	httpClient          *http.Client
	endpointURL         string
	apiKey              string
	dataSource          string
	writeToFile         bool
	activePatients      map[string]models.PatientStream
	patientBatches      map[string]*PatientBatch
	vitalsCache         map[string]*CachedVitals
	lastStreamedTimes   map[string]int64
	activePatientsMu    sync.RWMutex
	patientBatchesMu    sync.Mutex
	vitalsCacheMu       sync.RWMutex
	lastStreamedTimesMu sync.Mutex
}

func NewBeltProcessor(repo *database.Repository, endpointURL, apiKey, dataSource string, writeToFile bool) (*BeltProcessor, error) {
	p := &BeltProcessor{
		db:                repo,
		httpClient:        &http.Client{Timeout: 15 * time.Second},
		endpointURL:       endpointURL,
		apiKey:            apiKey,
		dataSource:        dataSource,
		writeToFile:       writeToFile,
		patientBatches:    make(map[string]*PatientBatch),
		activePatients:    make(map[string]models.PatientStream),
		vitalsCache:       make(map[string]*CachedVitals),
		lastStreamedTimes: make(map[string]int64),
	}

	if err := p.loadActivePatients(); err != nil {
		return nil, err
	}
	log.Printf("Service restored. Monitoring %d patients.", len(p.activePatients))
	return p, nil
}

func (p *BeltProcessor) RunHousekeepingCycle(ctx context.Context) {
	log.Println("Housekeeping cycle started. Will update DB every 1 minute.")
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Housekeeping cycle stopping.")
			return
		case <-ticker.C:
			now := time.Now().Unix()

			var patientsToPrune []string
			p.activePatientsMu.RLock()
			for patientID, patient := range p.activePatients {
				if patient.Status == "stopped" {
					patientsToPrune = append(patientsToPrune, patientID)
				}
			}
			p.activePatientsMu.RUnlock()

			if len(patientsToPrune) > 0 {
				p.activePatientsMu.Lock()
				for _, patientID := range patientsToPrune {
					delete(p.activePatients, patientID)
				}
				p.activePatientsMu.Unlock()
			}

			p.lastStreamedTimesMu.Lock()
			updatesToProcess := make(map[string]int64, len(p.lastStreamedTimes))
			for patientID, ts := range p.lastStreamedTimes {
				updatesToProcess[patientID] = ts
			}
			p.lastStreamedTimes = make(map[string]int64)
			p.lastStreamedTimesMu.Unlock()

			if len(updatesToProcess) > 0 {
				if err := p.db.BatchUpdateLastStreamedTime(updatesToProcess); err != nil {
					log.Printf("ERROR during housekeeping DB update: %v", err)
				} else {
					log.Printf("Housekeeping: Updated last streamed time for %d patients in DB.", len(updatesToProcess))
				}
			}

			recentVitals := make(map[string]string)
			var clearedDeviceIDs []string
			p.vitalsCacheMu.Lock()
			for patientID, vitals := range p.vitalsCache {
				if now-vitals.LastUpdated > 300 {
					delete(p.vitalsCache, patientID)
					if vitals.DeviceID != "" {
						clearedDeviceIDs = append(clearedDeviceIDs, vitals.DeviceID)
					}
				} else {
					recentVitals[patientID] = vitals.DeviceID
				}
			}
			p.vitalsCacheMu.Unlock()

			var report strings.Builder
			report.WriteString("\n--- Housekeeping Report ---\n")
			report.WriteString(fmt.Sprintf("%-15s | %-15s | %-10s | %-18s\n", "Patient", "Belt ID", "Streaming?", "Recent Vital Device?"))
			report.WriteString(strings.Repeat("-", 66) + "\n")

			p.activePatientsMu.RLock()
			if len(p.activePatients) == 0 {
				report.WriteString("No active patients being monitored.\n")
			} else {
				for patientID, patient := range p.activePatients {
					streamingStatus := "false"
					if _, ok := updatesToProcess[patientID]; ok {
						streamingStatus = "true"
					}
					vitalDevice := "none"
					if deviceID, ok := recentVitals[patientID]; ok {
						vitalDevice = deviceID
					}
					report.WriteString(fmt.Sprintf(
						"%-15s | %-15s | %-10s | %-18s\n",
						patientID,
						patient.DeviceID,
						streamingStatus,
						vitalDevice,
					))
				}
			}
			p.activePatientsMu.RUnlock()

			if len(patientsToPrune) > 0 {
				report.WriteString(fmt.Sprintf("Pruned %d stopped patient(s) from cache.\n", len(patientsToPrune)))
			}

			clearedDevicesStr := "none"
			if len(clearedDeviceIDs) > 0 {
				clearedDevicesStr = fmt.Sprintf("[%s]", strings.Join(clearedDeviceIDs, ", "))
			}
			report.WriteString(fmt.Sprintf("Stale Vitals Caches Cleared (%d): %s\n", len(clearedDeviceIDs), clearedDevicesStr))
			report.WriteString("------------------------------------------------------------------")
			log.Println(report.String())
		}
	}
}

func (p *BeltProcessor) RouteVitalsMessage(msgValue []byte) {
	var genericMsg map[string]interface{}
	if err := json.Unmarshal(msgValue, &genericMsg); err != nil {
		log.Printf("Error unmarshalling message for routing: %v", err)
		return
	}

	if _, ok := genericMsg["bp"]; ok {
		p.HandleBPSPO2Message(msgValue)
	} else if _, ok := genericMsg["spo2"]; ok {
		p.HandleBPSPO2Message(msgValue)
	} else if _, ok := genericMsg["ECG_CH_A"]; ok {
		p.HandleECGMessage(msgValue)
	} else {
		log.Printf("Unknown message type received on vitals topic, ignoring. Message: %s", string(msgValue))
	}
}

func (p *BeltProcessor) loadActivePatients() error {
	patients, err := p.db.GetActivePatients()
	if err != nil {
		return err
	}
	p.activePatientsMu.Lock()
	defer p.activePatientsMu.Unlock()
	for _, patient := range patients {
		p.activePatients[patient.PatientID] = patient
	}
	return nil
}

func (p *BeltProcessor) HandleBPSPO2Message(msgValue []byte) {
	var msg models.BPSPO2Message
	if err := json.Unmarshal(msgValue, &msg); err != nil {
		log.Printf("Error unmarshalling BP/SPO2 message: %v. Raw message: %s", err, string(msgValue))
		return
	}
	if msg.PatientID == "" {
		return
	}

	p.activePatientsMu.RLock()
	_, isActive := p.activePatients[msg.PatientID]
	p.activePatientsMu.RUnlock()

	if !isActive {
		log.Printf("[%s] Received vitals for an inactive patient. Caching data.", msg.PatientID)
	}

	p.vitalsCacheMu.Lock()
	defer p.vitalsCacheMu.Unlock()

	vitals, exists := p.vitalsCache[msg.PatientID]
	if !exists {
		vitals = &CachedVitals{}
	}

	vitals.DeviceID = msg.DeviceID
	var updateDetails []string
	updateDetails = append(updateDetails, fmt.Sprintf("SPO2=%d", msg.SPO2.Spo2), fmt.Sprintf("PR=%d", msg.SPO2.PulseRate))
	vitals.SPO2 = models.VitalSign{IsValid: true, Value: msg.SPO2.Spo2, Timestamp: msg.EpochTime}
	vitals.PR = models.VitalSign{IsValid: true, Value: msg.SPO2.PulseRate, Timestamp: msg.EpochTime}

	if msg.BP.BPSystolic != 0 {
		updateDetails = append(updateDetails, fmt.Sprintf("BP=%d/%d", msg.BP.BPSystolic, msg.BP.BPDiastolic))
		vitals.BP = models.BloodPressure{
			IsValid:   true,
			Sys:       msg.BP.BPSystolic,
			Dia:       msg.BP.BPDiastolic,
			Timestamp: msg.EpochTime,
		}
	}

	vitals.LastUpdated = time.Now().Unix()
	p.vitalsCache[msg.PatientID] = vitals

	log.Printf("[%s] Updated vitals cache: %s", msg.PatientID, strings.Join(updateDetails, ", "))
}

// MODIFIED: Removed the noisy log message for inactive ECG packets.
func (p *BeltProcessor) HandleECGMessage(msgValue []byte) {
	var msg models.ECGMessage
	if err := json.Unmarshal(msgValue, &msg); err != nil {
		log.Printf("Error unmarshalling ECG message: %v. Raw message: %s", err, string(msgValue))
		return
	}
	p.activePatientsMu.RLock()
	patientStream, isActive := p.activePatients[msg.PatientID]
	p.activePatientsMu.RUnlock()

	// Silently discard data for inactive or stopped patients to avoid log spam.
	if !isActive || patientStream.Status == "stopped" {
		return
	}

	if msg.Discharge {
		traceID := fmt.Sprintf("%s-%d", msg.PatientID, msg.PacketNo)
		p.processAndSendBatch(msg.PatientID, &PatientBatch{Messages: []*models.ECGMessage{&msg}}, traceID)
		return
	}

	p.patientBatchesMu.Lock()
	defer p.patientBatchesMu.Unlock()
	batch, exists := p.patientBatches[msg.PatientID]
	if !exists {
		batch = &PatientBatch{Messages: make([]*models.ECGMessage, 0, chunkSize)}
		p.patientBatches[msg.PatientID] = batch
	}
	batch.Messages = append(batch.Messages, &msg)
	if len(batch.Messages) >= chunkSize {
		lastMessage := batch.Messages[len(batch.Messages)-1]
		traceID := fmt.Sprintf("%s-%d", msg.PatientID, lastMessage.PacketNo)
		go p.processAndSendBatch(msg.PatientID, batch, traceID)
		delete(p.patientBatches, msg.PatientID)
	}
}

func (p *BeltProcessor) processAndSendBatch(patientID string, batch *PatientBatch, traceID string) {
	if len(batch.Messages) == 0 {
		return
	}
	var output models.PresensePayload
	var metadataSet bool
	for _, payload := range batch.Messages {
		if !metadataSet && payload.FacilityID != "" {
			output.DeviceType = payload.DeviceType
			output.PatientRef = fmt.Sprintf("%s-%s-%s", payload.FacilityID, payload.PatientID, payload.AdmissionID)
			output.FacilityID = payload.FacilityID
			output.PatchID = payload.DeviceID
			output.PatientName = payload.PatientName
			output.Timestamp = payload.CurrentTimestamp
			output.Gender = payload.Gender
			output.Age = payload.Age
			output.BiosensorStatus = "Connected"
			output.Source = p.dataSource
			metadataSet = true
		}
		sensorItem := models.SensorDataItem{
			SEQ:        payload.PacketNo,
			Timestamp:  payload.CurrentTimestamp,
			HR:         payload.HR,
			RhythmType: payload.RhythmType,
			RR:         payload.RR,
		}
		output.SensorData = append(output.SensorData, sensorItem)
	}

	if !metadataSet {
		return
	}
	p.vitalsCacheMu.RLock()
	cachedData, found := p.vitalsCache[patientID]
	if found {
		output.BP = cachedData.BP
		output.SPO2 = cachedData.SPO2
		output.PR = cachedData.PR
	}
	p.vitalsCacheMu.RUnlock()
	lastMessage := batch.Messages[len(batch.Messages)-1]
	output.ArrythmiaData = []models.ArrythmiaItem{{RhythmType: lastMessage.RhythmType}}
	output.EWS = map[string]interface{}{"ewsInfo": map[string]interface{}{}}
	jsonData, err := json.Marshal(output)
	if err != nil {
		log.Printf("[%s] Error marshalling processed data for patient %s: %v", traceID, patientID, err)
		return
	}
	if p.writeToFile {
		p.saveToFile(patientID, output.PatchID, output.Timestamp, jsonData)
	}
	if p.endpointURL != "" && p.apiKey != "" {
		p.sendToApi(patientID, jsonData, traceID)
	}
}

func (p *BeltProcessor) saveToFile(patientID, patchID string, timestamp int64, jsonData []byte) {
	dirPath := filepath.Join("../processed_data", patientID)
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		log.Printf("Error creating directory %s: %v", dirPath, err)
		return
	}
	filename := fmt.Sprintf("%s_%d.json", patchID, timestamp)
	fullPath := filepath.Join(dirPath, filename)
	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, jsonData, "", "  "); err != nil {
		log.Printf("Could not prettify JSON for file: %v", err)
		return
	}
	if err := os.WriteFile(fullPath, prettyJSON.Bytes(), 0644); err != nil {
		log.Printf("Error writing to file %s: %v", fullPath, err)
	}
}

func (p *BeltProcessor) sendToApi(patientID string, jsonData []byte, traceID string) {
	req, err := http.NewRequest("POST", p.endpointURL, bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("[%s] Error creating API request: %v", traceID, err)
		return
	}
	authHeader := "Bearer " + p.apiKey
	req.Header.Set("Authorization", authHeader)
	req.Header.Set("Content-Type", "application/json")
	resp, err := p.httpClient.Do(req)
	if err != nil {
		log.Printf("[%s] Error sending data to Presense API: %v", traceID, err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		log.Printf("[%s] Presense API returned non-success status: %s", traceID, resp.Status)
	} else {
		log.Printf("[%s] Successfully sent batch to Presense API. Status: %s", traceID, resp.Status)
		p.lastStreamedTimesMu.Lock()
		p.lastStreamedTimes[patientID] = time.Now().Unix()
		p.lastStreamedTimesMu.Unlock()
	}
}

func (p *BeltProcessor) HandleSvcStartMessage(payload []byte) {
	var msg models.SvcStartPayload
	if err := json.Unmarshal(payload, &msg); err != nil {
		return
	}
	if msg.DeviceType != "BIOSENSOR_NEXUS" {
		return
	}
	p.activePatientsMu.Lock()
	defer p.activePatientsMu.Unlock()
	if err := p.db.StartMonitoring(msg.PatientID, msg.FacilityID, msg.PatchID); err != nil {
		log.Printf("DB Error starting monitoring for patient %s: %v", msg.PatientID, err)
	} else {
		p.activePatients[msg.PatientID] = models.PatientStream{
			PatientID:  msg.PatientID,
			DeviceID:   msg.PatchID,
			FacilityID: msg.FacilityID,
			StartTime:  time.Now().Unix(),
			Status:     "running",
		}
		log.Printf("✅ Started monitoring patient: %s", msg.PatientID)
	}
}

func (p *BeltProcessor) HandleSvcActionMessage(payload []byte) {
	var msg models.SvcActionPayload
	if err := json.Unmarshal(payload, &msg); err != nil {
		return
	}
	if msg.Action == "stop" {
		p.activePatientsMu.Lock()
		defer p.activePatientsMu.Unlock()
		var patientIDToStop string
		for id, stream := range p.activePatients {
			if stream.DeviceID == msg.PatchID {
				patientIDToStop = id
				break
			}
		}
		if patientIDToStop == "" {
			return
		}
		if err := p.db.StopMonitoring(patientIDToStop); err != nil {
			log.Printf("DB Error stopping monitoring for patient %s: %v", patientIDToStop, err)
		} else {
			if patient, ok := p.activePatients[patientIDToStop]; ok {
				patient.Status = "stopped"
				now := time.Now().Unix()
				patient.EndTime = &now
				p.activePatients[patientIDToStop] = patient
			}
			p.patientBatchesMu.Lock()
			delete(p.patientBatches, patientIDToStop)
			p.patientBatchesMu.Unlock()
			log.Printf("🛑 Stopped monitoring patient: %s", patientIDToStop)
		}
	}
}
