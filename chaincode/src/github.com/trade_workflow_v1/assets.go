package main

type Resource struct{
	TypeOfResource		string		`json:"typeofresource"`
	Quantity			string		`json:"quantity"`
	Beneficiary			string		`json:"beneficiary"`
	Source				string		`json:"source"`
	Status				string		`json:"status"`
	Carrier				string		`json:"carrier"`
	Description			string 		`json:"description"`
	CurrentLocation		string		`json:"location"`
}


type MoveInShelter struct{
	Capacity			int			`json:"capacity"`
	Address				string		`josn:"address"`
	Food				bool		`json:"food"`
	People				[]string	`json:"people"`
}


type GovernmentRequest struct{

}

type Victim struct{
	Reliefcamp			string		`json:"reliefcamp"`
	HealthCondition		string		`json:"health"`
	Details				Participant `json:"details"`
	NumRequests			int			`json:"NumRequests"`

}

type Volunteer struct{
	Localhub 			string 		`json:"localhub"`
	Details				Participant `json:"details"`
}

type Participant struct{
	Email				string 		`json:"email"`
	Location			string		`json:"location"`
	Description			string		`json:"description"`
}

type VictimRequest struct{
	Email				string		`json:"email"`
	RequestId			string		`json:"requestid"`
	RequestTo			string		`json:"requestedto"`
	Status				string		`json:"status"`
	Need				string		`json:"need"`
	Quantity			string		`json:"quantity"`
	Description			string		`json:"description"`
}
