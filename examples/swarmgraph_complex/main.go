package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	swarmgo "github.com/prathyushnallamothu/swarmgo"
	"github.com/prathyushnallamothu/swarmgo/llm"
)

// Medical clinic workflow example using the new LangGraph-inspired workflow system

// FetchPatientRecord simulates retrieving patient records from a database
func FetchPatientRecord(args map[string]interface{}, contextVariables map[string]interface{}) swarmgo.Result {
	patientID, ok := args["patient_id"].(string)
	if !ok {
		return swarmgo.Result{
			Success: false,
			Data:    "Error: patient_id is required",
		}
	}

	// Simulate database lookup
	patientData := map[string]interface{}{
		"id":        patientID,
		"name":      "John Doe",
		"age":       45,
		"gender":    "Male",
		"allergies": []string{"Penicillin"},
		"history": []map[string]interface{}{
			{
				"date":      "2023-10-15",
				"diagnosis": "Hypertension",
				"treatment": "Prescribed Lisinopril 10mg",
			},
			{
				"date":      "2024-01-22",
				"diagnosis": "Seasonal Allergies",
				"treatment": "Prescribed Cetirizine 10mg",
			},
		},
	}

	// Store in context
	if contextVariables != nil {
		contextVariables["patient"] = patientData
	}

	return swarmgo.Result{
		Success: true,
		Data:    fmt.Sprintf("Retrieved patient record for %s (John Doe, 45, Male)", patientID),
	}
}

// ScheduleAppointment simulates scheduling a patient appointment
func ScheduleAppointment(args map[string]interface{}, contextVariables map[string]interface{}) swarmgo.Result {
	patientID, ok1 := args["patient_id"].(string)
	doctor, ok2 := args["doctor"].(string)
	date, ok3 := args["date"].(string)
	time, ok4 := args["time"].(string)

	if !ok1 || !ok2 || !ok3 || !ok4 {
		return swarmgo.Result{
			Success: false,
			Data:    "Error: patient_id, doctor, date and time are all required",
		}
	}

	appointment := map[string]interface{}{
		"patient_id": patientID,
		"doctor":     doctor,
		"date":       date,
		"time":       time,
		"status":     "scheduled",
	}

	if contextVariables != nil {
		contextVariables["appointment"] = appointment
	}

	return swarmgo.Result{
		Success: true,
		Data:    fmt.Sprintf("Appointment scheduled with Dr. %s on %s at %s", doctor, date, time),
	}
}

// OrderLabTest simulates ordering lab tests
func OrderLabTest(args map[string]interface{}, contextVariables map[string]interface{}) swarmgo.Result {
	patientID, ok1 := args["patient_id"].(string)
	testType, ok2 := args["test_type"].(string)

	if !ok1 || !ok2 {
		return swarmgo.Result{
			Success: false,
			Data:    "Error: patient_id and test_type are required",
		}
	}

	labOrder := map[string]interface{}{
		"patient_id": patientID,
		"test_type":  testType,
		"status":     "ordered",
		"order_date": time.Now().Format("2006-01-02"),
	}

	if contextVariables != nil {
		contextVariables["lab_order"] = labOrder
	}

	return swarmgo.Result{
		Success: true,
		Data:    fmt.Sprintf("Lab test '%s' ordered for patient %s", testType, patientID),
	}
}

// PrescribeMedication simulates prescribing medication
func PrescribeMedication(args map[string]interface{}, contextVariables map[string]interface{}) swarmgo.Result {
	patientID, ok1 := args["patient_id"].(string)
	medication, ok2 := args["medication"].(string)
	dosage, ok3 := args["dosage"].(string)
	frequency, ok4 := args["frequency"].(string)

	if !ok1 || !ok2 || !ok3 || !ok4 {
		return swarmgo.Result{
			Success: false,
			Data:    "Error: patient_id, medication, dosage and frequency are required",
		}
	}

	// Check for allergies
	if patientData, ok := contextVariables["patient"].(map[string]interface{}); ok {
		if allergies, ok := patientData["allergies"].([]string); ok {
			for _, allergy := range allergies {
				if strings.Contains(strings.ToLower(medication), strings.ToLower(allergy)) {
					return swarmgo.Result{
						Success: false,
						Data:    fmt.Sprintf("WARNING: Patient is allergic to %s", allergy),
					}
				}
			}
		}
	}

	prescription := map[string]interface{}{
		"patient_id": patientID,
		"medication": medication,
		"dosage":     dosage,
		"frequency":  frequency,
		"date":       time.Now().Format("2006-01-02"),
	}

	if contextVariables != nil {
		contextVariables["prescription"] = prescription
	}

	return swarmgo.Result{
		Success: true,
		Data:    fmt.Sprintf("Prescribed %s %s %s for patient %s", medication, dosage, frequency, patientID),
	}
}

func main() {
	// Load environment variables
	godotenv.Load()

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable is required")
	}

	// Create all the agents for the workflow nodes

	// Create a graph for our medical clinic workflow
	builder := swarmgo.NewGraphBuilder("Medical Clinic Workflow", "Workflow for patient processing in a medical clinic")

	// 1. Receptionist Agent - Handles patient intake and scheduling
	receptionistAgent := &swarmgo.Agent{
		Name: "Receptionist",
		Instructions: `You are a medical clinic receptionist. 
Your responsibilities:
1. Greet patients and collect their basic information
2. Verify patient records in the system
3. Schedule appointments with appropriate doctors
4. Direct patients to the right department

Always be professional, courteous, and efficient. Collect patient ID when possible.`,
		Model: "gpt-4",
		Functions: []swarmgo.AgentFunction{
			{
				Name:        "fetch_patient_record",
				Description: "Retrieve a patient's medical record",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"patient_id": map[string]interface{}{
							"type":        "string",
							"description": "Patient identification number",
						},
					},
					"required": []interface{}{"patient_id"},
				},
				Function: FetchPatientRecord,
			},
			{
				Name:        "schedule_appointment",
				Description: "Schedule an appointment for a patient",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"patient_id": map[string]interface{}{
							"type":        "string",
							"description": "Patient identification number",
						},
						"doctor": map[string]interface{}{
							"type":        "string",
							"description": "Doctor's name",
						},
						"date": map[string]interface{}{
							"type":        "string",
							"description": "Appointment date (YYYY-MM-DD)",
						},
						"time": map[string]interface{}{
							"type":        "string",
							"description": "Appointment time (HH:MM)",
						},
					},
					"required": []interface{}{"patient_id", "doctor", "date", "time"},
				},
				Function: ScheduleAppointment,
			},
		},
	}

	// 2. Nurse Agent - Takes vitals and prepares patients
	nurseAgent := &swarmgo.Agent{
		Name: "Nurse",
		Instructions: `You are a clinic nurse.
Your responsibilities:
1. Take patient vitals (blood pressure, temperature, etc.)
2. Record patient symptoms and concerns
3. Prepare patients for examination
4. Assist doctors during procedures

Be caring, attentive, and thorough in your assessments.`,
		Model: "gpt-4",
		Functions: []swarmgo.AgentFunction{
			{
				Name:        "fetch_patient_record",
				Description: "Retrieve a patient's medical record",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"patient_id": map[string]interface{}{
							"type":        "string",
							"description": "Patient identification number",
						},
					},
					"required": []interface{}{"patient_id"},
				},
				Function: FetchPatientRecord,
			},
			{
				Name:        "record_vitals",
				Description: "Record patient vitals",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"patient_id": map[string]interface{}{
							"type":        "string",
							"description": "Patient identification number",
						},
						"temperature": map[string]interface{}{
							"type":        "string",
							"description": "Body temperature in F or C",
						},
						"blood_pressure": map[string]interface{}{
							"type":        "string",
							"description": "Blood pressure reading (systolic/diastolic)",
						},
						"heart_rate": map[string]interface{}{
							"type":        "string",
							"description": "Heart rate in BPM",
						},
					},
					"required": []interface{}{"patient_id"},
				},
				Function: func(args map[string]interface{}, contextVars map[string]interface{}) swarmgo.Result {
					// Simple function to record vitals
					patientID := args["patient_id"].(string)

					vitals := map[string]interface{}{}
					for key, value := range args {
						if key != "patient_id" {
							vitals[key] = value
						}
					}

					if contextVars != nil {
						contextVars["vitals"] = vitals
					}

					return swarmgo.Result{
						Success: true,
						Data:    fmt.Sprintf("Recorded vitals for patient %s", patientID),
					}
				},
			},
		},
	}

	// 3. Doctor Agent - Diagnoses patients and prescribes treatment
	doctorAgent := &swarmgo.Agent{
		Name: "Doctor",
		Instructions: `You are a medical doctor at a clinic.
Your responsibilities:
1. Review patient history and current symptoms
2. Perform examinations and make diagnoses
3. Order appropriate tests and interpret results
4. Prescribe medications and treatments
5. Provide medical advice and follow-up plans

Be thorough, accurate, and compassionate in your care.`,
		Model: "gpt-4",
		Functions: []swarmgo.AgentFunction{
			{
				Name:        "fetch_patient_record",
				Description: "Retrieve a patient's medical record",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"patient_id": map[string]interface{}{
							"type":        "string",
							"description": "Patient identification number",
						},
					},
					"required": []interface{}{"patient_id"},
				},
				Function: FetchPatientRecord,
			},
			{
				Name:        "order_lab_test",
				Description: "Order laboratory tests for a patient",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"patient_id": map[string]interface{}{
							"type":        "string",
							"description": "Patient identification number",
						},
						"test_type": map[string]interface{}{
							"type":        "string",
							"description": "Type of test to order (e.g., blood panel, urinalysis)",
						},
					},
					"required": []interface{}{"patient_id", "test_type"},
				},
				Function: OrderLabTest,
			},
			{
				Name:        "prescribe_medication",
				Description: "Prescribe medication for a patient",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"patient_id": map[string]interface{}{
							"type":        "string",
							"description": "Patient identification number",
						},
						"medication": map[string]interface{}{
							"type":        "string",
							"description": "Medication name",
						},
						"dosage": map[string]interface{}{
							"type":        "string",
							"description": "Dosage amount",
						},
						"frequency": map[string]interface{}{
							"type":        "string",
							"description": "How often to take the medication",
						},
					},
					"required": []interface{}{"patient_id", "medication", "dosage", "frequency"},
				},
				Function: PrescribeMedication,
			},
		},
	}

	// 4. Lab Technician Agent - Processes lab tests
	labTechAgent := &swarmgo.Agent{
		Name: "LabTechnician",
		Instructions: `You are a medical laboratory technician.
Your responsibilities:
1. Process lab test orders
2. Collect specimens when necessary
3. Run laboratory tests
4. Record and report test results
5. Maintain lab equipment and standards

Be precise, methodical, and attentive to detail.`,
		Model: "gpt-3.5-turbo",
		Functions: []swarmgo.AgentFunction{
			{
				Name:        "fetch_patient_record",
				Description: "Retrieve a patient's medical record",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"patient_id": map[string]interface{}{
							"type":        "string",
							"description": "Patient identification number",
						},
					},
					"required": []interface{}{"patient_id"},
				},
				Function: FetchPatientRecord,
			},
			{
				Name:        "process_lab_test",
				Description: "Process a laboratory test",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"patient_id": map[string]interface{}{
							"type":        "string",
							"description": "Patient identification number",
						},
						"test_type": map[string]interface{}{
							"type":        "string",
							"description": "Type of test",
						},
						"results": map[string]interface{}{
							"type":        "string",
							"description": "Test results",
						},
					},
					"required": []interface{}{"patient_id", "test_type", "results"},
				},
				Function: func(args map[string]interface{}, contextVars map[string]interface{}) swarmgo.Result {
					// Process lab test and record results
					patientID := args["patient_id"].(string)
					testType := args["test_type"].(string)
					results := args["results"].(string)

					testResults := map[string]interface{}{
						"patient_id": patientID,
						"test_type":  testType,
						"results":    results,
						"date":       time.Now().Format("2006-01-02"),
						"status":     "completed",
					}

					if contextVars != nil {
						contextVars["test_results"] = testResults
					}

					return swarmgo.Result{
						Success: true,
						Data:    fmt.Sprintf("Processed %s test for patient %s with results: %s", testType, patientID, results),
					}
				},
			},
		},
	}

	// 5. Pharmacist Agent - Dispenses medications and provides instructions
	pharmacistAgent := &swarmgo.Agent{
		Name: "Pharmacist",
		Instructions: `You are a clinic pharmacist.
Your responsibilities:
1. Review medication orders for accuracy
2. Check for drug interactions and contradictions
3. Prepare and dispense medications
4. Provide medication information to patients
5. Ensure proper medication management

Be meticulous, knowledgeable, and patient-focused.`,
		Model: "gpt-3.5-turbo",
		Functions: []swarmgo.AgentFunction{
			{
				Name:        "fetch_patient_record",
				Description: "Retrieve a patient's medical record",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"patient_id": map[string]interface{}{
							"type":        "string",
							"description": "Patient identification number",
						},
					},
					"required": []interface{}{"patient_id"},
				},
				Function: FetchPatientRecord,
			},
			{
				Name:        "dispense_medication",
				Description: "Dispense medication for a patient",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"patient_id": map[string]interface{}{
							"type":        "string",
							"description": "Patient identification number",
						},
						"medication": map[string]interface{}{
							"type":        "string",
							"description": "Medication name",
						},
						"instructions": map[string]interface{}{
							"type":        "string",
							"description": "Instructions for taking the medication",
						},
					},
					"required": []interface{}{"patient_id", "medication", "instructions"},
				},
				Function: func(args map[string]interface{}, contextVars map[string]interface{}) swarmgo.Result {
					// Dispense medication
					patientID := args["patient_id"].(string)
					medication := args["medication"].(string)
					instructions := args["instructions"].(string)

					dispensed := map[string]interface{}{
						"patient_id":   patientID,
						"medication":   medication,
						"instructions": instructions,
						"date":         time.Now().Format("2006-01-02"),
						"status":       "dispensed",
					}

					if contextVars != nil {
						contextVars["dispensed_medication"] = dispensed
					}

					return swarmgo.Result{
						Success: true,
						Data:    fmt.Sprintf("Dispensed %s for patient %s with instructions: %s", medication, patientID, instructions),
					}
				},
			},
		},
	}

	// Now let's build the actual workflow graph

	// Add all agent nodes to the graph
	builder.WithAgent("reception", "Receptionist", receptionistAgent)
	builder.WithAgent("nurse", "Nurse", nurseAgent)
	builder.WithAgent("doctor", "Doctor", doctorAgent)
	builder.WithAgent("lab", "Lab Technician", labTechAgent)
	builder.WithAgent("pharmacy", "Pharmacist", pharmacistAgent)

	// Create a router node to determine where patient should go next
	routerCondition := func(state swarmgo.GraphState) (swarmgo.NodeID, error) {
		// Get the last message content to determine routing
		messagesRaw, ok := state[swarmgo.MessageKey]
		if !ok || messagesRaw == nil {
			return "reception", nil // Default to reception
		}

		var messages []llm.Message
		messagesData, _ := json.Marshal(messagesRaw)
		if err := json.Unmarshal(messagesData, &messages); err != nil {
			return "reception", nil
		}

		if len(messages) == 0 {
			return "reception", nil
		}

		// Get last message content
		content := strings.ToLower(messages[len(messages)-1].Content)

		// Check for exit keywords first
		if strings.Contains(content, "checkout") ||
			strings.Contains(content, "done") ||
			strings.Contains(content, "complete") ||
			strings.Contains(content, "finished") ||
			strings.Contains(content, "thank you") {
			return "exit", nil // Exit the workflow
		}

		// Count messages to prevent infinite loops
		msgCount := len(messages)
		if msgCount > 20 {
			// If we've had too many exchanges, force exit
			return "exit", nil
		}

		// Track visit progress in state
		visits := make(map[string]int)
		if visitsRaw, exists := state["node_visits"]; exists {
			if v, ok := visitsRaw.(map[string]int); ok {
				visits = v
			}
		}

		// Route based on keywords
		var nextNode swarmgo.NodeID

		if strings.Contains(content, "appointment") || strings.Contains(content, "schedule") {
			nextNode = "reception"
		} else if strings.Contains(content, "test") || strings.Contains(content, "lab") {
			nextNode = "lab"
		} else if strings.Contains(content, "medication") || strings.Contains(content, "prescription") {
			nextNode = "pharmacy"
		} else if strings.Contains(content, "symptom") || strings.Contains(content, "pain") ||
			strings.Contains(content, "doctor") || strings.Contains(content, "diagnosis") {

			// Check if vitals have been taken
			_, hasVitals := state["vitals"]
			if !hasVitals {
				nextNode = "nurse" // See nurse first for vitals
			} else {
				nextNode = "doctor" // Already has vitals, see doctor
			}
		} else {
			// Progress through typical flow if no specific keywords
			if visits["reception"] == 0 {
				nextNode = "reception"
			} else if visits["nurse"] == 0 {
				nextNode = "nurse"
			} else if visits["doctor"] == 0 {
				nextNode = "doctor"
			} else if visits["pharmacy"] == 0 {
				nextNode = "pharmacy"
			} else {
				// Completed all departments, go to exit
				nextNode = "exit"
			}
		}

		// Increment visit count for the next node
		visits[string(nextNode)]++

		// Store updated visit counts in state
		newState := state.Clone()
		newState["node_visits"] = visits

		// Prevent infinite loops by limiting visits to each node
		if visits[string(nextNode)] > 3 {
			// If we've visited this node too many times, force progression
			if nextNode == "reception" {
				nextNode = "nurse"
			} else if nextNode == "nurse" {
				nextNode = "doctor"
			} else if nextNode == "doctor" {
				nextNode = "pharmacy"
			} else if nextNode == "pharmacy" || nextNode == "lab" {
				nextNode = "exit"
			}
		}

		return nextNode, nil
	}

	// Create special node for tracking conversation state
	builder.WithNode("router", "Patient Router", func(ctx context.Context, state swarmgo.GraphState) (swarmgo.GraphState, error) {
		// Router modifies state to track visits
		newState := state.Clone()

		// Initialize visit tracking if it doesn't exist
		if _, exists := newState["node_visits"]; !exists {
			newState["node_visits"] = make(map[string]int)
		}

		// Get current node from graph structure
		currentNode, ok := newState["current_node"].(string)
		if ok && currentNode != "" {
			// Update visit count for current node
			if visits, ok := newState["node_visits"].(map[string]int); ok {
				visits[currentNode]++
				newState["node_visits"] = visits
			}
		}

		// Set a flag to indicate this is not the first visit to router
		newState["router_visited"] = true

		return newState, nil
	})
	// Create special nodes for workflow control
	builder.WithNode("intake", "Patient Intake", func(ctx context.Context, state swarmgo.GraphState) (swarmgo.GraphState, error) {
		// This node just initializes the state with patient greeting
		newState := state.Clone()

		// Add initial message if none exists
		messagesRaw, exists := state[swarmgo.MessageKey]
		if !exists || messagesRaw == nil {
			newState[swarmgo.MessageKey] = []llm.Message{
				{
					Role:    llm.RoleUser,
					Content: "Hello, I'd like to see a doctor today. My name is John Doe.",
				},
			}
		}

		return newState, nil
	})

	builder.WithNode("exit", "Checkout Process", func(ctx context.Context, state swarmgo.GraphState) (swarmgo.GraphState, error) {
		// Process checkout, billing, etc.
		newState := state.Clone()

		// Get messages
		messagesRaw, exists := state[swarmgo.MessageKey]
		if !exists || messagesRaw == nil {
			return newState, nil
		}

		var messages []llm.Message
		messagesData, _ := json.Marshal(messagesRaw)
		if err := json.Unmarshal(messagesData, &messages); err != nil {
			return newState, nil
		}

		// Add checkout message
		messages = append(messages, llm.Message{
			Role:    llm.RoleAssistant,
			Content: "Thank you for visiting our clinic today. Your visit has been processed and any applicable charges have been sent to billing.",
		})

		newState[swarmgo.MessageKey] = messages
		newState["checkout_complete"] = true

		return newState, nil
	})

	builder.WithNode("router", "Patient Router", func(ctx context.Context, state swarmgo.GraphState) (swarmgo.GraphState, error) {
		// Router just passes state through unchanged
		return state, nil
	})

	// Add conditional edges for the router
	builder.WithConditionalEdge("router", "reception", routerCondition)
	builder.WithConditionalEdge("router", "nurse", routerCondition)
	builder.WithConditionalEdge("router", "doctor", routerCondition)
	builder.WithConditionalEdge("router", "lab", routerCondition)
	builder.WithConditionalEdge("router", "pharmacy", routerCondition)

	// Set up the workflow connections
	builder.WithEdge("intake", "reception")

	// All nodes can route to other departments as needed
	builder.WithEdge("reception", "router")
	builder.WithEdge("nurse", "router")
	builder.WithEdge("doctor", "router")
	builder.WithEdge("lab", "router")
	builder.WithEdge("pharmacy", "exit")

	// Set entry and exit points
	builder.WithEntryPoint("intake")
	builder.WithExitPoint("exit")

	// Build the complete graph
	graph := builder.Build()

	// Create a graph runner to execute the workflow
	runner := swarmgo.NewGraphRunner()
	runner.RegisterGraph(graph)

	// Initialize state with API key and model info
	initialState := swarmgo.GraphState{
		"api_key":  apiKey,
		"provider": string(llm.OpenAI),
	}

	// Execute the workflow (this would typically be triggered by an API endpoint or scheduler)
	fmt.Println("Starting Medical Clinic Workflow simulation...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	finalState, err := runner.ExecuteGraph(ctx, graph.ID, initialState)
	if err != nil {
		log.Fatalf("Error executing workflow: %v", err)
	}

	// Print final results
	fmt.Println("\nWorkflow completed successfully!")
	fmt.Println("Final state contains:")

	for key, value := range finalState {
		if key == swarmgo.MessageKey {
			fmt.Println("\nConversation history:")
			var messages []llm.Message
			messagesData, _ := json.Marshal(value)
			json.Unmarshal(messagesData, &messages)

			for i, msg := range messages {
				switch msg.Role {
				case llm.RoleUser:
					fmt.Printf("%d. Patient: %s\n", i+1, msg.Content)
				case llm.RoleAssistant:
					fmt.Printf("%d. Clinic Staff: %s\n", i+1, msg.Content)
				case llm.RoleFunction:
					fmt.Printf("%d. System: [%s] %s\n", i+1, msg.Name, msg.Content)
				}
			}
		} else if key == "checkout_complete" {
			fmt.Println("\nCheckout status: Complete")
		} else if strings.HasPrefix(string(key), "var_") {
			fmt.Printf("\nVariable %s: %v\n", strings.TrimPrefix(string(key), "var_"), value)
		} else if key == "patient" || key == "prescription" || key == "vitals" ||
			key == "test_results" || key == "dispensed_medication" {
			fmt.Printf("\n%s: %v\n", key, value)
		}
	}
}
