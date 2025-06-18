package events
//TODO: check if I need to convert received datetime information in the Form in the Request type from string to Time.time. I don't think this is necessary when using built in Marshalling and Unmarshalling functions. There is a MarshalText encoder associated with the time.Time type.
import(
	"context"
	"fmt"
	"time"
	"net/http"
	"encoding/json"
	"github.com/jackc/pgx/v5"
	"io/ioutil"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"github.com/nadams128/oatnet/server/auth"
)

type eventStruct struct {
	Name string `json:"eventName"`
	Datetime time.Time `json:"datetime"`
	Location string `json:"location"`
	Description string `json:"description"`

}

func RequestHandler(w http.ResponseWriter, r *http.Request) {
	conn, _ := pgx.Connect(context.Background(), "postgres://oatnet:password@127.0.0.1/oatnet")
	defer conn.Close(context.Background())
	switch r.Method {
		case "GET":
			getEvents(w, r, conn)
		case "POST":
			postEvents(w, r, conn)
		case "DELETE":
			deleteEvents(w, r, conn)
		case "OPTIONS":
			optionsEvents(w, r)
	}
}

//This only shows events occuring in the future. I may delete events that have already occured later.
func getEvents(w http.ResponseWriter, r *http.Request, conn *pgx.Conn) {
	w.Header().Set("Access-Control-Allow-Origin","*")
	formParseError := r.ParseForm()
	if formParseError != nil {
		fmt.Println(formParseError)
	}
	var requestedRows pgx.Rows
	sessionIDHeader := r.Header["Sessionid"]
	var sessionID string
	if sessionIDHeader != nil {
		sessionID = sessionIDHeader[0]
	}
	read, _ := auth.CheckPermissions(sessionID, conn)
	if read {
		filter, filterParam := r.Form["filter"]
		var selectErr error
		currentTime := time.Now()
		if filterParam {
			switch filter[0] {
			case "all":
				requestedRows, selectErr = conn.Query(context.Background(), "SELECT * FROM events WHERE eventDatetime > $1 ORDER BY eventDatetime;", currentTime)
		} else {
			requestedRows, selectErr = conn.Query(context.Background(), "SELECT * FROM events ORDER BY eventDatetime;")
		}
		if selectErr != nil {
			fmt.Println(selectErr)
		}
		var eventName string
		var datetime time.Time
		var location string
		var description string
		responseList, _ := pgx.CollectRows(requestedRows, func(row pgx.CollectableRow) (eventStruct, error) {
			err := row.Scan(&eventName, &datetime, &location, &description)
			return eventStruct{eventName, datetime, location, description}, err
		})
		var jsonResponseList, marshalErr = json.Marshal(responseList)
		if marshalErr != nil {
			fmt.Println(marshalErr)
		}
		w.Write(jsonResponseList)
	}
}
func postEvents(w http.ResponseWriter, r *http.Request, conn *pgx.Conn) {
	var responseMessage string = "Event creation failed! >:"
	sessionIDHeader := r.Header["Sessionid"]
	var sessionID string
	if sessionIDHeader != nil {
		sessionID = sessionIDHeader[0]
	}
	_, write := auth.CheckPermissions(sessionID, conn)
	if write {
		var event eventStruct
		receivedBytes, readErr := ioutil.ReadAll(r.Body)
		if readErr != nil {
			fmt.Println(readErr)
		}
		unmarshalErr := json.Unmarshal(receivedBytes, &event)
		if unmarshalErr != nil {
			fmt.Println(unmarshalErr)
		}
		
		requestedEvent, _ := conn.Query(context.Background(), "SELECT * FROM events  WHERE eventName=$1 AND eventDatetime=$2;", cases.Title(language.AmericanEnglish).String(event.Name), event.Datetime)
		var requestedEventExists bool = requestedEvent.Next()
		requestedEvent.Close()
		if requestedEventExists {
			_,updateErr := conn.Exec(context.Background(), "UPDATE events  SET location=$1, description=$2 WHERE eventName=$3 AND eventDatetime=$4;", event.Location, event.Description, cases.Title(language.AmericanEnglish).String(event.Name), event.Datetime)
			responseMessage = "Event updated! :>"
			if updateErr != nil {
				fmt.Println(updateErr)
			}
		} else {
			_,insertErr := conn.Exec(context.Background(), "INSERT INTO events VALUES($1,$2,$3,$4);", cases.Title(language.AmericanEnglish).String(event.Name), event.Datetime, event.Location, event.Description)
			responseMessage = "Event added! :>"
			if insertErr != nil {
				fmt.Println(insertErr)
			}
		}
	}
	var jsonResponseMessage, marshalErr = json.Marshal(responseMessage)
	if marshalErr != nil {
		fmt.Println(marshalErr)
	}
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Write(jsonResponseMessage)
}

func deleteEvents(w http.ResponseWriter, r *http.Request, conn *pgx.Conn) {
	formParseError := r.ParseForm()
	if formParseError != nil {
		fmt.Println("formParseError ", formParseError)
	}
	var responseMessage string = "Delete failed! >:"
	sessionIDHeader := r.Header["Sessionid"]
	var sessionID string
	if sessionIDHeader != nil {
		sessionID = sessionIDHeader[0]
	}
	_, write := auth.CheckPermissions(sessionID, conn)
	if write {
		var event eventStruct
		receivedBytes, readErr := ioutil.ReadAll(r.Body)
		if readErr != nil {
			fmt.Println(readErr)
		}
		unmarshalErr := json.Unmarshal(receivedBytes, &event)
		if unmarshalErr != nil {
			fmt.Println(unmarshalErr)
		}
		
		if unmarshalErr == nil {
			_, deleteErr := conn.Exec(context.Background(), "DELETE FROM events WHERE eventName=$1 AND eventDatetime = $2;", cases.Title(language.AmericanEnglish).String(event.Name), event.Datetime) 
			if deleteErr!=nil {
				fmt.Println("deleteErr ", deleteErr)
			} else {
				responseMessage = "Item deleted! :>"
			}
		}
	}
	var jsonResponseMessage, marshalErr = json.Marshal(responseMessage)
	if marshalErr != nil {
		fmt.Println("marshalErr ", marshalErr)
	}
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Write(jsonResponseMessage)
}

func optionsEvents(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, sessionID")
	w.Header().Set("Access-Control-Allow-Methods", "OPTIONS, POST, DELETE")
}
