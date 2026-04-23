This is the system design for ticketmaster 

functional requirements:
view an event
book an event
search for an event

non functional requirements:
low latency search
consistency while booking
availability while view/search event


stack:
postgres for db
redis for caching


endpoints:

POST /events
- body {
	name
	total_tickets
}

GET /events
GET /events/:eventId

POST /book
- body {
	event_id
	user_id
	quantity
}

This is just v1. In v2, we shall implement search, reserve and confirm