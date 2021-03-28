package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"reflect"
	"strings"
	"sync"

	"github.com/gorilla/mux"
)

var result map[string]interface{}

type eventData struct {
	mu           sync.Mutex
	scheduleInfo map[string]interface{}
}

func GlobalNotReadyHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/global" {
		http.Error(w, "404 not found", http.StatusNotFound)
		return
	}
	if r.Method != "GET" {
		http.Error(w, "Method is not Supported.", http.StatusNotFound)
		return
	}
	fmt.Fprintf(w, "No Events have been created yet!!! Page Cannot render anything")
}

const mutexLocked = 1

func MutexLocked(m *sync.Mutex) bool {
	state := reflect.ValueOf(m).Elem().FieldByName("state")
	return state.Int()&mutexLocked == mutexLocked
}

func (eventDataInstance *eventData) slotBooking(wg *sync.WaitGroup, input_timestamp string, input_name string, input_event string, w http.ResponseWriter) {
	defer wg.Done()
	eventDataInstance.mu.Lock()
	returnValue := ""
	// I am Locking the eventData variable
	fmt.Println("Before booking: ", eventDataInstance.scheduleInfo)
	_, found := eventDataInstance.scheduleInfo[input_timestamp]
	fmt.Println("For ", input_name, "FOund: ", found)
	if found == false {
		fmt.Println("User ", input_name, " saw that event is free to book")
		map_internal := map[string]interface{}{"user": input_name,
			"event": input_event, "blockedUsers": ""}
		eventDataInstance.scheduleInfo[input_timestamp] = map_internal
		fmt.Println(eventDataInstance)
		fmt.Println("User ", input_name, " successfully booked the slot")
		returnValue = "User " + input_name + " successfully booked the slot"
	} else {
		//blockedUsers := ""
		internal_map := eventDataInstance.scheduleInfo[input_timestamp].(map[string]interface{})
		eventUsers, _ := json.Marshal(internal_map["user"])
		getUserInfo := strings.Split(string(eventUsers), ",")
		status := "no"
		fmt.Println(string(input_name), getUserInfo[0])

		if string(input_name) == strings.ReplaceAll(getUserInfo[0], `"`, "") {
			status = "organizer"
			fmt.Println(status, ";;", eventDataInstance)
		} else {
			for _, element := range getUserInfo {
				if string(input_name) == strings.ReplaceAll(element, `"`, "") {
					status = "participant"
					break
				}
			}
		}
		if status == "no" {
			internalMap := eventDataInstance.scheduleInfo[input_timestamp].(map[string]interface{})
			blockedUsers, _ := json.Marshal(internalMap["blockedUsers"])
			getBlockedUsers := strings.Split(string(blockedUsers), ",")
			var map_update map[string]interface{}
			if len(getBlockedUsers) > 0 {
				map_update = map[string]interface{}{"user": internalMap["user"],
					"event": internalMap["event"], "blockedUsers": input_name}

			} else {
				map_update = map[string]interface{}{"user": internalMap["user"],
					"event": internalMap["event"], "blockedUsers": string(blockedUsers) + "," + input_name}
			}

			eventDataInstance.scheduleInfo[input_timestamp] = map_update
			fmt.Println("User ", input_name, " is unable to book the slot as it was already booked")
			returnValue = "User " + input_name + " is unable to book the slot as it was already booked"
		} else if status == "organizer" {
			fmt.Println("You have already booked the slot!")
			returnValue = "You have already booked the slot!"
		} else if status == "participant" {
			fmt.Println("Your event has been booked by your organizer: ", getUserInfo[0])
			returnValue = "Your event has been booked by your organizer: " + getUserInfo[0]
		}
	}
	eventDataInstance.mu.Unlock()
	fmt.Println("mutex locked = ", MutexLocked(&eventDataInstance.mu))
	// return returnValue
	fmt.Fprintf(w, returnValue)
}
func (eventDataInstance *eventData) bookHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		fmt.Fprintf(w, "ParseForm() err: %v", err)
		return
	}
	input_name := r.FormValue("name")
	input_time := r.FormValue("time")
	input_date := r.FormValue("date")
	input_event := r.FormValue("event")
	input_check_box := r.Form["enable_participants"]
	fmt.Println(input_check_box)
	fmt.Println(reflect.TypeOf(input_check_box))
	if len(input_check_box) > 0 {
		if input_check_box[0] == "on" {
			participants := r.FormValue("participants")
			input_name = input_name + "," + participants
		}
	}
	fmt.Println("INput name finally: ", input_name)
	input_timestamp := input_time + "_" + input_date
	fmt.Println(input_timestamp)
	wg := new(sync.WaitGroup)
	wg.Add(1)
	go eventDataInstance.slotBooking(wg, input_timestamp, input_name, input_event, w)

	wg.Wait()
}
func (eventDataInstance *eventData) viewBookings(input_name string) map[string]interface{} {
	fmt.Println("Entered view page", "input_user", input_name)
	fmt.Println("mutex locked = ", MutexLocked(&eventDataInstance.mu))
	eventDataInstance.mu.Lock()
	fmt.Println("Entered view page")
	var userData eventData
	userData.scheduleInfo = eventDataInstance.scheduleInfo
	userSchedule := userData.scheduleInfo
	fmt.Println(userSchedule)
	deleteKey := ""
	for k := range userSchedule {
		if deleteKey != "" {
			delete(userSchedule, deleteKey)
		}
		internal_map := userSchedule[k].(map[string]interface{})
		eventUsers, _ := json.Marshal(internal_map["user"])
		getUserInfo := strings.Split(string(eventUsers), ",")
		blocked_eventUsers, _ := json.Marshal(internal_map["blockedUsers"])
		blocked_getUserInfo := strings.Split(string(blocked_eventUsers), ",")
		status := "no"
		fmt.Println(string(input_name), getUserInfo[0])

		if string(input_name) == strings.ReplaceAll(getUserInfo[0], `"`, "") {
			status = "organizer"
			fmt.Println(status, ";;", userSchedule)
		} else {
			for _, element := range getUserInfo {
				if string(input_name) == strings.ReplaceAll(element, `"`, "") {
					status = "participant"
					break
				}
			}
			for _, blocked_element := range blocked_getUserInfo {
				if strings.ReplaceAll(blocked_element, `"`, "") == input_name+" and xxxxSecret-Will-Not-Be-Displayedxxxxx" {
					status = "blocked"
					break
				} else {
					fmt.Println("delsss: ", string(input_name), strings.ReplaceAll(blocked_element, `"`, ""))
					if string(input_name) == strings.ReplaceAll(blocked_element, `"`, "") {
						status = "blocked"
						map_update := map[string]interface{}{"user": "xxxxSecret-Will-Not-Be-Displayedxxxxx",
							"event": "xxxxSecret-Will-Not-Be-Displayedxxxxx", "blockedUsers": input_name + " and xxxxSecret-Will-Not-Be-Displayedxxxxx"}
						userSchedule[k] = map_update
						break
					}
				}

			}

		}
		if status == "no" {
			deleteKey = k
		} else {
			deleteKey = ""
		}
		// Similar logic to find blocked users
		fmt.Println("del: ", deleteKey)
	}
	fmt.Println(userSchedule)
	fmt.Println("delete key finally: ", deleteKey)
	if deleteKey != "" {
		delete(userSchedule, deleteKey)
	}
	eventDataInstance.mu.Unlock()
	return userSchedule
}
func (eventDataInstance *eventData) viewScheduleHandler(w http.ResponseWriter, r *http.Request) {
	input_name := mux.Vars(r)["user"]
	fmt.Println(eventDataInstance.viewBookings(input_name))
	userScheduleInfo, _ := json.Marshal(eventDataInstance.viewBookings(input_name))
	fmt.Fprintf(w, string(userScheduleInfo))
}
func (eventDataInstance *eventData) viewAllSchedulesHandler(w http.ResponseWriter, r *http.Request) {
	eventDataInstance.mu.Lock()
	totalInfo := eventDataInstance.scheduleInfo
	eventDataInstance.mu.Unlock()
	delete(totalInfo, "timestamp")
	allUserScheduleInfo, _ := json.Marshal(totalInfo)
	fmt.Fprintf(w, string(allUserScheduleInfo))
}
func main() {
	filePath := "data.json"
	fmt.Printf("// reading file %s\n", filePath)
	file, err1 := ioutil.ReadFile(filePath)
	if err1 != nil {
		fmt.Printf("// error while reading file %s\n", filePath)
		fmt.Printf("File error: %v\n", err1)
		os.Exit(1)
	}
	err2 := json.Unmarshal([]byte(file), &result)
	if err2 != nil {
		fmt.Println("error:", err2)
		os.Exit(1)
	}
	eventDataInstance := eventData{scheduleInfo: result}

	fileServer := http.FileServer(http.Dir("./static"))
	schedulerRouter := mux.NewRouter().StrictSlash(true)
	schedulerRouter.Handle("/", fileServer)
	schedulerRouter.HandleFunc("/book", eventDataInstance.bookHandler)
	schedulerRouter.HandleFunc("/view/{user}", eventDataInstance.viewScheduleHandler)
	schedulerRouter.HandleFunc("/f10", eventDataInstance.viewAllSchedulesHandler)
	log.Fatal(http.ListenAndServe(":9000", schedulerRouter))
}
