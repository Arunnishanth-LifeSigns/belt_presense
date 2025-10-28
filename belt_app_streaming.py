import threading
import time
import json
import random
import string
import paho.mqtt.client as mqtt
from kafka import KafkaProducer
from kafka.errors import NoBrokersAvailable

# --- Configuration ---
MQTT_BROKER = "localhost"
MQTT_PORT = 1883
MQTT_START_TOPIC = "arrhythmia/svc_start"
MQTT_ACTION_TOPIC = "arrhythmia/svc_action"

KAFKA_BROKER = "localhost:9092"
KAFKA_VITALS_TOPIC = "patient-vitals-data-topic"

running_belts = {}
running_bpspo2_sims = {}
# NEW: Dictionary to track lonely BP/SPO2 simulators
running_lonely_bpspo2_sims = {}

class BPSPO2Simulator:
    def __init__(self, patient_info, kafka_producer):
        self.kafka_producer = kafka_producer
        self.patient_info = patient_info
        self.device_id = "SIM-LEPU-" + ''.join(random.choices(string.ascii_uppercase + string.digits, k=4))
        self.stop_event = threading.Event()
        self.thread = threading.Thread(target=self.run, daemon=True)

    def _generate_bpspo2_payload(self, include_bp=False):
        payload = {
            "patientId": self.patient_info['patient_id'],
            "facilityId": self.patient_info['facility_id'],
            "admissionId": self.patient_info['admission_id'],
            "deviceID": self.device_id,
            "epochTime": int(time.time()),
            "spo2": { "spo2": random.randint(95, 99), "pulseRate": random.randint(60, 100) }
        }
        if include_bp:
            payload["bp"] = { "bpSystolic": random.randint(115, 135), "bpDiastolic": random.randint(75, 90) }
        return payload

    def run(self):
        print(f"\n[*] Starting BP/SPO2 monitoring for Patient: {self.patient_info['patient_id']} ({self.device_id})")
        
        print(f"    -> ({self.patient_info['patient_id']}) Sending initial BP and SPO2 reading...")
        initial_payload = self._generate_bpspo2_payload(include_bp=True)
        self.kafka_producer.send(
            KAFKA_VITALS_TOPIC,
            key=self.patient_info['patient_id'].encode('utf-8'),
            value=json.dumps(initial_payload).encode('utf-8')
        )

        cycle_counter = 0
        while not self.stop_event.is_set():
            time.sleep(1)
            cycle_counter += 1
            payload = None
            if cycle_counter % 180 == 0:
                print(f"    -> ({self.patient_info['patient_id']}) Sending BP and SPO2...")
                payload = self._generate_bpspo2_payload(include_bp=True)
            elif cycle_counter % 30 == 0:
                print(f"    -> ({self.patient_info['patient_id']}) Sending SPO2 only...")
                payload = self._generate_bpspo2_payload(include_bp=False)
            if payload:
                self.kafka_producer.send(
                    KAFKA_VITALS_TOPIC,
                    key=self.patient_info['patient_id'].encode('utf-8'),
                    value=json.dumps(payload).encode('utf-8')
                )
        print(f"[!] BP/SPO2 monitoring stopped for: {self.patient_info['patient_id']}")

    def start(self):
        self.thread.start()
    def stop(self):
        self.stop_event.set()

class BeltSimulator:
    def __init__(self, mqtt_client, kafka_producer):
        self.mqtt_client = mqtt_client
        self.kafka_producer = kafka_producer
        self.patient_id = f"SIM-PAT-{''.join(random.choices(string.digits, k=4))}"
        self.facility_id = "SIM-HOSP"
        self.admission_id = f"ADM-{''.join(random.choices(string.digits, k=6))}"
        self.device_id = "SIM-BELT-" + ''.join(random.choices(string.ascii_uppercase + string.digits, k=4))
        self.patient_name = f"Simulated Patient {random.randint(100, 999)}"
        self.packet_counter = 1
        self.stop_event = threading.Event()
        self.thread = threading.Thread(target=self.run, daemon=True)

    def _generate_ecg_payload(self):
        payload = {
            "facilityId": self.facility_id, "patientId": self.patient_id, "admissionId": self.admission_id,
            "deviceId": self.device_id, "deviceType": "Belt", "patientName": self.patient_name,
            "gender": random.choice(["Male", "Female"]), "age": random.randint(30, 80),
            "currentTimestamp": int(time.time() * 1000), "packetNo": self.packet_counter,
            "ECG_CH_A": [round(random.uniform(-0.5, 0.5), 4) for _ in range(125)],
            "HR": random.randint(65, 95), "RR": random.randint(16, 22), "rhythmType": "SR"
        }
        self.packet_counter += 1
        return payload

    def run(self):
        print(f"\n[*] Starting belt: {self.device_id} for Patient: {self.patient_id}")
        self.send_start_command()
        print(f"    -> Streaming ECG data to Kafka topic '{KAFKA_VITALS_TOPIC}'...")
        while not self.stop_event.is_set():
            payload = self._generate_ecg_payload()
            self.kafka_producer.send(
                KAFKA_VITALS_TOPIC,
                key=self.patient_id.encode('utf-8'),
                value=json.dumps(payload).encode('utf-8')
            )
            time.sleep(1)
        print(f"[!] Stream stopped for belt: {self.device_id}")

    def send_start_command(self):
        payload = {
            "patchId": self.device_id, "facilityId": self.facility_id, "serviceId": "arrhythmia",
            "providerId": "presense", "patientId": self.patient_id, "deviceType": "BIOSENSOR_NEXUS"
        }
        self.mqtt_client.publish(MQTT_START_TOPIC, json.dumps(payload))
        print(f"    -> Sent START command to MQTT topic '{MQTT_START_TOPIC}'")

    def send_stop_command(self):
        payload = { "patchId": self.device_id, "action": "stop" }
        self.mqtt_client.publish(MQTT_ACTION_TOPIC, json.dumps(payload))
        print(f"\n[!] Sent STOP command for {self.device_id} to MQTT topic '{MQTT_ACTION_TOPIC}'")

    def start(self):
        self.thread.start()
    def stop(self):
        self.send_stop_command()
        self.stop_event.set()

# --- UI Functions ---
def start_new_belt(mqtt_client, kafka_producer):
    belt_sim = BeltSimulator(mqtt_client, kafka_producer)
    belt_sim.start()
    running_belts[belt_sim.device_id] = belt_sim

# MODIFIED: Stop function now handles both Belts and lonely BP/SPO2 devices
def stop_running_stream():
    all_streams = []
    for belt_sim in running_belts.values():
        all_streams.append({"type": "Belt", "sim": belt_sim, "id": belt_sim.device_id, "patient": belt_sim.patient_id})
    for bp_sim in running_lonely_bpspo2_sims.values():
        all_streams.append({"type": "Lonely BP/SPO2", "sim": bp_sim, "id": bp_sim.device_id, "patient": bp_sim.patient_info['patient_id']})

    if not all_streams:
        print("\nNo streams are currently running.")
        return

    print("\n--- Running Streams ---")
    for i, stream in enumerate(all_streams):
        print(f"  {i+1}: {stream['type']} - {stream['id']} (Patient: {stream['patient']})")
    
    try:
        choice = int(input("Enter the number of the stream to stop (or 0 to cancel): "))
        if 0 < choice <= len(all_streams):
            stream_to_stop = all_streams[choice - 1]
            stream_to_stop['sim'].stop()
            if stream_to_stop['type'] == "Belt":
                del running_belts[stream_to_stop['id']]
                if stream_to_stop['id'] in running_bpspo2_sims:
                    running_bpspo2_sims[stream_to_stop['id']].stop()
                    del running_bpspo2_sims[stream_to_stop['id']]
            else: # Lonely BP/SPO2
                del running_lonely_bpspo2_sims[stream_to_stop['id']]
        else:
            print("Cancelled or invalid number.")
    except ValueError:
        print("Invalid input. Please enter a number.")

def add_bpspo2_monitoring(kafka_producer):
    if not running_belts:
        print("\nNo belts are currently running. Start a belt first.")
        return
    
    print("\n--- Select Patient to Add BP/SPO2 Monitoring ---")
    belt_list = list(running_belts.values())
    for i, sim in enumerate(belt_list):
        status = "(Already monitoring)" if sim.device_id in running_bpspo2_sims else ""
        print(f"  {i+1}: {sim.device_id} (Patient: {sim.patient_id}) {status}")
        
    try:
        choice = int(input("Enter the number of the patient (or 0 to cancel): "))
        if 0 < choice <= len(belt_list):
            belt_to_monitor = belt_list[choice - 1]
            if belt_to_monitor.device_id in running_bpspo2_sims:
                print("This patient already has BP/SPO2 monitoring active.")
                return
            patient_info = {
                'patient_id': belt_to_monitor.patient_id,
                'facility_id': belt_to_monitor.facility_id,
                'admission_id': belt_to_monitor.admission_id
            }
            bpspo2_sim = BPSPO2Simulator(patient_info, kafka_producer)
            bpspo2_sim.start()
            running_bpspo2_sims[belt_to_monitor.device_id] = bpspo2_sim
        else:
            print("Cancelled or invalid number.")
    except ValueError:
        print("Invalid input. Please enter a number.")

# MODIFIED: Renamed function and updated to show lonely devices
def list_running_services():
    print("\n--- Running Services ---")
    if not running_belts and not running_lonely_bpspo2_sims:
        print("  No services are currently running.")
        return
        
    for device_id, sim in running_belts.items():
        bpspo2_status = "YES" if device_id in running_bpspo2_sims else "NO"
        print(f"  - ECG Belt: {device_id} (Patient: {sim.patient_id}) | Associated BP/SPO2: {bpspo2_status}")
        
    for device_id, sim in running_lonely_bpspo2_sims.items():
        print(f"  - Lonely BP/SPO2: {device_id} (Patient: {sim.patient_info['patient_id']})")

# NEW: Function to start a BP/SPO2 stream for a non-monitored patient
def start_lonely_bpspo2(kafka_producer):
    print("\nStarting a BP/SPO2 stream for a new, non-monitored patient...")
    patient_info = {
        'patient_id': f"SIM-LONE-{''.join(random.choices(string.digits, k=4))}",
        'facility_id': "SIM-HOSP",
        'admission_id': f"ADM-{''.join(random.choices(string.digits, k=6))}"
    }
    bpspo2_sim = BPSPO2Simulator(patient_info, kafka_producer)
    bpspo2_sim.start()
    running_lonely_bpspo2_sims[bpspo2_sim.device_id] = bpspo2_sim

# NEW: Function to remove BP/SPO2 monitoring from an active belt
def remove_bpspo2_monitoring():
    if not running_bpspo2_sims:
        print("\nNo patients have associated BP/SPO2 monitoring to remove.")
        return

    monitored_belts = []
    for belt_id, bpspo2_sim in running_bpspo2_sims.items():
        if belt_id in running_belts:
             monitored_belts.append({"belt_id": belt_id, "patient_id": running_belts[belt_id].patient_id})

    if not monitored_belts:
        print("\nNo patients have associated BP/SPO2 monitoring to remove.")
        return

    print("\n--- Select Patient to Remove BP/SPO2 Monitoring From ---")
    for i, belt_info in enumerate(monitored_belts):
        print(f"  {i+1}: {belt_info['belt_id']} (Patient: {belt_info['patient_id']})")

    try:
        choice = int(input("Enter the number of the patient (or 0 to cancel): "))
        if 0 < choice <= len(monitored_belts):
            belt_to_modify = monitored_belts[choice - 1]
            belt_id = belt_to_modify['belt_id']
            
            print(f"\nStopping BP/SPO2 monitoring for patient {belt_to_modify['patient_id']}...")
            running_bpspo2_sims[belt_id].stop()
            del running_bpspo2_sims[belt_id]
        else:
            print("Cancelled or invalid number.")
    except ValueError:
        print("Invalid input. Please enter a number.")

def main_ui():
    try:
        mqtt_client = mqtt.Client(mqtt.CallbackAPIVersion.VERSION2)
        mqtt_client.connect(MQTT_BROKER, MQTT_PORT, 60)
        mqtt_client.loop_start()
        print(f"✅ Connected to MQTT Broker at {MQTT_BROKER}:{MQTT_PORT}")
    except ConnectionRefusedError:
        print(f"❌ FATAL: Connection to MQTT broker refused. Please ensure it's running.")
        return

    try:
        kafka_producer = KafkaProducer(bootstrap_servers=KAFKA_BROKER)
        print(f"✅ Connected to Kafka Broker at {KAFKA_BROKER}")
    except NoBrokersAvailable:
        print(f"❌ FATAL: Could not connect to Kafka broker. Please ensure it's running.")
        mqtt_client.loop_stop()
        mqtt_client.disconnect()
        return

    # MODIFIED: Updated menu with new options
    while True:
        print("\n===== Go Application Belt Simulator =====")
        print("  1: Start a new ECG belt stream")
        print("  2: Add associated BP/SPO2 monitoring to a patient")
        print("  3: Start a lonely BP/SPO2 stream (no ECG)")
        print("  4: Remove associated BP/SPO2 monitoring")
        print("  5: Stop a running stream (ECG or lonely BP/SPO2)")
        print("  6: List all running services")
        print("  7: Exit")
        choice = input(">> Enter your choice: ")
        if choice == '1':
            start_new_belt(mqtt_client, kafka_producer)
        elif choice == '2':
            add_bpspo2_monitoring(kafka_producer)
        elif choice == '3':
            start_lonely_bpspo2(kafka_producer)
        elif choice == '4':
            remove_bpspo2_monitoring()
        elif choice == '5':
            stop_running_stream()
        elif choice == '6':
            list_running_services()
        elif choice == '7':
            print("Shutting down... Stopping all streams.")
            for sim in running_belts.values():
                sim.stop()
            for sim in running_bpspo2_sims.values():
                sim.stop()
            # MODIFIED: Ensure lonely sims are also stopped
            for sim in running_lonely_bpspo2_sims.values():
                sim.stop()
            time.sleep(1)
            break
        else:
            print("Invalid choice, please try again.")

    print("Flushing Kafka messages...")
    kafka_producer.flush()
    kafka_producer.close()
    mqtt_client.loop_stop()
    mqtt_client.disconnect()
    print("Simulator shutdown complete.")

if __name__ == "__main__":
    main_ui()