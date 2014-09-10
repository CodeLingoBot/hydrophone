package api

import (
	"log"
	"net/http"
	"net/url"

	"./../clients"
	"./../models"
	"github.com/gorilla/mux"
)

type (
	Api struct {
		Store     clients.StoreClient
		notifier  clients.Notifier
		templates models.EmailTemplate
		//shoreline *shoreline.ShorelineClient
		Config Config
	}
	Config struct {
		ServerSecret string                `json:"serverSecret"` //used for services
		Templates    *models.EmailTemplate `json:"emailTemplates"`
	}
	// this just makes it easier to bind a handler for the Handle function
	varsHandler func(http.ResponseWriter, *http.Request, map[string]string)
)

const (
	TP_SESSION_TOKEN         = "x-tidepool-session-token"
	STATUS_ERR_SENDING_EMAIL = "Error sending email"
	STATUS_OK                = "OK"
)

func InitApi(cfg Config, store clients.StoreClient, ntf clients.Notifier) *Api {

	return &Api{
		Store:    store,
		Config:   cfg,
		notifier: ntf,
		//shoreline: sl,
	}
}

func (a *Api) SetHandlers(prefix string, rtr *mux.Router) {

	rtr.HandleFunc("/status", a.GetStatus).Methods("GET")

	rtr.Handle("/emailtoaddress/{type}/{address}", varsHandler(a.EmailAddress)).Methods("GET", "POST")

	// POST /confirm/send/signup/:userid
	// POST /confirm/send/forgot/:useremail
	// POST /confirm/send/invite/:userid
	send := rtr.PathPrefix("/send").Subrouter()
	send.Handle("/signup/{userid}", varsHandler(a.Dummy)).Methods("POST")
	send.Handle("/forgot/{useremail}", varsHandler(a.Dummy)).Methods("POST")
	send.Handle("/invite/{userid}", varsHandler(a.Dummy)).Methods("POST")

	// POST /confirm/resend/signup/:userid
	rtr.Handle("/resend/signup/{userid}", varsHandler(a.Dummy)).Methods("POST")

	// PUT /confirm/accept/signup/:userid/:confirmationID
	// PUT /confirm/accept/forgot/
	// PUT /confirm/accept/invite/:userid/:invited_by
	accept := rtr.PathPrefix("/accept").Subrouter()
	accept.Handle("/signup/{userid}/{confirmationid}",
		varsHandler(a.Dummy)).Methods("PUT")
	accept.Handle("/forgot", varsHandler(a.Dummy)).Methods("PUT")
	accept.Handle("/invite/{userid}/{invitedby}",
		varsHandler(a.Dummy)).Methods("PUT")

	// GET /confirm/signup/:userid
	// GET /confirm/invite/:userid
	rtr.Handle("/signup/{userid}", varsHandler(a.Dummy)).Methods("GET")
	rtr.Handle("/invite/{userid}", varsHandler(a.Dummy)).Methods("GET")

	// GET /confirm/invitations/:userid
	rtr.Handle("/invitations/{userid}", varsHandler(a.Dummy)).Methods("GET")

	// PUT /confirm/dismiss/invite/:userid/:invited_by
	// PUT /confirm/dismiss/signup/:userid
	dismiss := rtr.PathPrefix("/dismiss").Subrouter()
	dismiss.Handle("/invite/{userid}/{invitedby}",
		varsHandler(a.Dummy)).Methods("PUT")
	dismiss.Handle("/signup/{userid}",
		varsHandler(a.Dummy)).Methods("PUT")

	// DELETE /confirm/:userid/invited/:invited_address
	// DELETE /confirm/signup/:userid
	rtr.Handle("/{userid}/invited/{invited_address}", varsHandler(a.Dummy)).Methods("DELETE")
	rtr.Handle("/signup/{userid}", varsHandler(a.Dummy)).Methods("DELETE")
}

func (h varsHandler) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	h(res, req, vars)
}

func (a *Api) Dummy(res http.ResponseWriter, req *http.Request, vars map[string]string) {
	log.Printf("dummy() ignored request %s %s", req.Method, req.URL)
	res.WriteHeader(http.StatusOK)
}

func (a *Api) GetStatus(res http.ResponseWriter, req *http.Request) {
	if err := a.Store.Ping(); err != nil {
		log.Println("Error getting status", err)
		res.WriteHeader(http.StatusInternalServerError)
		res.Write([]byte(err.Error()))
		return
	}
	res.WriteHeader(http.StatusOK)
	res.Write([]byte(STATUS_OK))
	return
}

func (a *Api) EmailAddress(res http.ResponseWriter, req *http.Request, vars map[string]string) {

	if token := req.Header.Get(TP_SESSION_TOKEN); token != "" {

		/* TODO: the actual token check once we have mocks in place
		if td := a.shoreline.CheckToken(token); td == nil {
			log.Println("bad token check ", td)
		}*/

		emailType := vars["type"]
		emailAddress, _ := url.QueryUnescape(vars["address"])

		if emailAddress != "" && emailType != "" {

			notification, err := models.NewEmailNotification(emailType, a.Config.Templates, emailAddress)

			if err != nil {
				log.Println("Error creating template ", err)
				res.Write([]byte(STATUS_ERR_SENDING_EMAIL))
				res.WriteHeader(http.StatusInternalServerError)
				return
			} else {
				if status, details := a.notifier.Send([]string{emailAddress}, "TODO", notification.Content); status != http.StatusOK {
					log.Printf("Issue sending email: Status [%d] Message [%s]", status, details)
					res.Write([]byte(STATUS_ERR_SENDING_EMAIL))
					res.WriteHeader(http.StatusInternalServerError)
					return
				} else {
					res.WriteHeader(http.StatusOK)
					return
				}
			}
		}
		res.WriteHeader(http.StatusBadRequest)
		return
	}
	res.WriteHeader(http.StatusUnauthorized)
	return
}
