package main

type Resource struct{
	TypeOfResource		string		`json:"typeofresource"`
	Quantity			string		`json:"quantity"`
	Beneficiary			string		`json:"beneficiary"`
	Source				string		`json:"source"`
	Status				string		`json:"status"`
	Carrier				string		`json:"carrier"`
	Description			string 		`json:"description"`
}

type Food struct{
	Res					Resource	`json:"resource"`
}

type Clothes struct{
	Res					Resource	`json:"resource"`
}

type MoveInShelter struct{
	Capacity			string		`json:"capacity"`
	Address				string		`josn:"address"`
	Food				bool		`json:"food"`
	Res					Resource	`json:"resource"`	
}

type Shelter struct{
	Res						Resource	`json:"resource"`
}

type MedicalKit struct{
	Res					Resource	`json:"resource"`
}

type GovernmentRequest struct{

}

type Victim struct{
	Reliefcamp			string		`json:"reliefcamp"`
	HealthCondition		string		`json:"health"`
}

type Volunteer struct{
	Localhub 			string 		`json:"localhub"`
}

type Participant struct{
	Email				string 		`json:"email"`
	Location			string		`json:"location"`
	Description			string		`json:"description"`
}