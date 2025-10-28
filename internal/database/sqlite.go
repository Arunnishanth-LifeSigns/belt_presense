package database

import (
	"database/sql"
	"log"
	"time"

	"belt-presense/internal/models"

	_ "github.com/mattn/go-sqlite3"
)

var istLocation *time.Location

// MODIFIED: Added slashes to the format string
const timeFormat = "02/01/2006 15:04:05.000"

func init() {
	loc, err := time.LoadLocation("Asia/Kolkata")
	if err != nil {
		log.Fatalf("FATAL: Failed to load IST timezone: %v", err)
	}
	istLocation = loc
}

type Repository struct {
	db *sql.DB
}

func NewRepository(dbPath string) (*Repository, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	repo := &Repository{db: db}
	if err := repo.initSchema(); err != nil {
		return nil, err
	}
	return repo, nil
}

func (r *Repository) initSchema() error {
	createSessionsTable := `
    CREATE TABLE IF NOT EXISTS monitoring_sessions (
        patient_id TEXT PRIMARY KEY,
        device_id TEXT,
        status TEXT NOT NULL,
        facility_id TEXT NOT NULL,
        start_time TEXT NOT NULL,
        end_time TEXT,
        last_streamed_time TEXT
    );`
	_, err := r.db.Exec(createSessionsTable)
	return err
}

func (r *Repository) StartMonitoring(patientID, facilityID, deviceID string) error {
	nowStr := time.Now().In(istLocation).Format(timeFormat)
	query := `INSERT OR REPLACE INTO monitoring_sessions (patient_id, device_id, status, facility_id, start_time, end_time, last_streamed_time) VALUES (?, ?, ?, ?, ?, NULL, NULL)`
	_, err := r.db.Exec(query, patientID, deviceID, "running", facilityID, nowStr)
	return err
}

func (r *Repository) StopMonitoring(patientID string) error {
	nowStr := time.Now().In(istLocation).Format(timeFormat)
	query := `UPDATE monitoring_sessions SET status = ?, end_time = ? WHERE patient_id = ?`
	_, err := r.db.Exec(query, "stopped", nowStr, patientID)
	return err
}

func (r *Repository) BatchUpdateLastStreamedTime(updates map[string]int64) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare("UPDATE monitoring_sessions SET last_streamed_time = ? WHERE patient_id = ?")
	if err != nil {
		tx.Rollback()
		return err
	}
	defer stmt.Close()

	for patientID, timestamp := range updates {
		timeStr := time.Unix(timestamp, 0).In(istLocation).Format(timeFormat)
		if _, err := stmt.Exec(timeStr, patientID); err != nil {
			log.Printf("Failed to update time for patient %s, rolling back transaction. Error: %v", patientID, err)
			tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

func (r *Repository) GetActivePatients() ([]models.PatientStream, error) {
	query := `SELECT patient_id, device_id, status, facility_id, start_time, end_time, last_streamed_time FROM monitoring_sessions WHERE status = 'running'`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var activePatients []models.PatientStream
	for rows.Next() {
		var patient models.PatientStream
		var startTimeStr string
		var endTimeStr, lastStreamedTimeStr sql.NullString

		if err := rows.Scan(
			&patient.PatientID,
			&patient.DeviceID,
			&patient.Status,
			&patient.FacilityID,
			&startTimeStr,
			&endTimeStr,
			&lastStreamedTimeStr,
		); err != nil {
			return nil, err
		}

		startTime, err := time.ParseInLocation(timeFormat, startTimeStr, istLocation)
		if err != nil {
			log.Printf("Warning: could not parse start_time '%s' from DB: %v", startTimeStr, err)
			continue
		}
		patient.StartTime = startTime.Unix()

		if endTimeStr.Valid {
			endTime, err := time.ParseInLocation(timeFormat, endTimeStr.String, istLocation)
			if err == nil {
				endTimeUnix := endTime.Unix()
				patient.EndTime = &endTimeUnix
			}
		}
		if lastStreamedTimeStr.Valid {
			lastStreamedTime, err := time.ParseInLocation(timeFormat, lastStreamedTimeStr.String, istLocation)
			if err == nil {
				lastStreamedTimeUnix := lastStreamedTime.Unix()
				patient.LastStreamedTime = &lastStreamedTimeUnix
			}
		}
		activePatients = append(activePatients, patient)
	}
	return activePatients, nil
}

func (r *Repository) Close() {
	r.db.Close()
}
