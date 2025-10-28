package models

// VitalsMessage is now split into ECGMessage and BPSPO2Message below.

// NEW: ECGMessage represents the data from the ECG device.
type ECGMessage struct {
	FacilityID       string    `json:"facilityId"`
	PatientID        string    `json:"patientId"`
	AdmissionID      string    `json:"admissionId"`
	DeviceID         string    `json:"deviceId"`
	DeviceType       string    `json:"deviceType"`
	PatientName      string    `json:"patientName"`
	Gender           string    `json:"gender"`
	Age              int       `json:"age"`
	CurrentTimestamp int64     `json:"currentTimestamp"`
	PacketNo         int64     `json:"packetNo"`
	ECG_CH_A         []float64 `json:"ECG_CH_A"`
	HR               int       `json:"HR"`
	RR               int       `json:"RR"`
	RhythmType       string    `json:"rhythmType"`
	Discharge        bool      `json:"Discharge,omitempty"`
}

// NEW: BPSPO2Message represents the data from the BP/SPO2 device.
type BPSPO2Message struct {
	PatientID   string `json:"patientId"`
	FacilityID  string `json:"facilityId"`
	AdmissionID string `json:"admissionId"`
	DeviceID    string `json:"deviceID"`
	EpochTime   int64  `json:"epochTime"`
	BP          struct {
		BPSystolic  int `json:"bpSystolic"`
		BPDiastolic int `json:"bpDiastolic"`
	} `json:"bp"`
	SPO2 struct {
		Spo2      int `json:"spo2"`
		PulseRate int `json:"pulseRate"`
	} `json:"spo2"`
}

// --- Structs for the outgoing Presense API Payload ---

type PresensePayload struct {
	DeviceType      string                 `json:"deviceType"`
	PatientRef      string                 `json:"patientRef"`
	FacilityID      string                 `json:"FacilityId"`
	PatchID         string                 `json:"PatchId"`
	PatientName     string                 `json:"PatientName"`
	Timestamp       int64                  `json:"TimeStamp"`
	BedID           string                 `json:"BedId"`
	Gender          string                 `json:"Gender"`
	Age             int                    `json:"Age"`
	BiosensorStatus string                 `json:"BiosensorStatus"`
	Source          string                 `json:"source"`
	SensorData      []SensorDataItem       `json:"SensorData"`
	SPO2            VitalSign              `json:"SPO2"`
	PR              VitalSign              `json:"PR"`
	BP              BloodPressure          `json:"BP"`
	ArrythmiaData   []ArrythmiaItem        `json:"ArrythmiaData"`
	EWS             map[string]interface{} `json:"ews"`
}

type SensorDataItem struct {
	ECG_CH_A    []float64 `json:"ECG_CH_A,omitempty"`
	ECG_CH_B    []int     `json:"ECG_CH_B,omitempty"`
	SEQ         int64     `json:"SEQ"`
	Timestamp   int64     `json:"TimeStamp"`
	HR          int       `json:"HR"`
	RhythmType  string    `json:"rhythmType"`
	RR          int       `json:"RR"`
	BODYTEMP    int       `json:"BODYTEMP,omitempty"`
	SKINTEMP    int       `json:"SKINTEMP,omitempty"`
	AMBTEMP_AVG int       `json:"AMBTEMP_AVG,omitempty"`
	TSECG       int64     `json:"TSECG,omitempty"`
}

type VitalSign struct {
	IsValid   bool  `json:"IsValid"`
	Value     int   `json:"Value,omitempty"`
	Timestamp int64 `json:"TimeStamp,omitempty"`
}

type BloodPressure struct {
	IsValid   bool  `json:"IsValid"`
	Sys       int   `json:"Sys,omitempty"`
	Dia       int   `json:"Dia,omitempty"`
	Timestamp int64 `json:"TimeStamp,omitempty"`
}

type ArrythmiaItem struct {
	RhythmType string `json:"rhythmType"`
}

// PatientStream represents a patient's monitoring session
type PatientStream struct {
	PatientID        string
	DeviceID         string // MODIFIED: Added DeviceID to link to patchId
	Status           string
	FacilityID       string
	StartTime        int64
	EndTime          *int64
	LastStreamedTime *int64
}

type SvcStartPayload struct {
	PatchID    string `json:"patchId"`
	FacilityID string `json:"facilityId"`
	ServiceID  string `json:"serviceId"`
	ProviderID string `json:"providerId"`
	PatientID  string `json:"patientId"`
	DeviceType string `json:"deviceType"`
}

type SvcActionPayload struct {
	PatchID string `json:"patchId"`
	Action  string `json:"action"`
}
