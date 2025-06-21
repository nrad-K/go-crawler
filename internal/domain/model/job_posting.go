package model

import (
	"time"

	"github.com/google/uuid"
)

type JobPostingArgs struct {
	ID           uuid.UUID
	Title        string
	CompanyName  string
	SummaryURL   string
	Location     Location
	Headquarters Location
	JobType      JobType
	Salary       Salary
	PostedAt     time.Time
	Details      JobPostingDetail
}

type JobPosting struct {
	id           uuid.UUID
	title        string
	companyName  string
	summaryURL   string
	location     Location
	headquarters Location
	jobType      JobType
	salary       Salary
	postedAt     time.Time
	details      JobPostingDetail
}

func NewJobPosting(args JobPostingArgs) JobPosting {
	return JobPosting{
		id:           args.ID,
		title:        args.Title,
		companyName:  args.CompanyName,
		summaryURL:   args.SummaryURL,
		location:     args.Location,
		headquarters: args.Headquarters,
		jobType:      args.JobType,
		salary:       args.Salary,
		postedAt:     args.PostedAt,
		details:      args.Details,
	}
}

func (j *JobPosting) ID() string {
	return j.id.String()
}

func (j *JobPosting) CompanyName() string {
	return j.companyName
}

func (j *JobPosting) Title() string {
	return j.title
}

func (j *JobPosting) SummaryURL() string {
	return j.summaryURL
}

func (j *JobPosting) Location() Location {
	return j.location
}

func (j *JobPosting) Headquarters() Location {
	return j.headquarters
}

func (j *JobPosting) JobType() JobType {
	return j.jobType
}

func (j *JobPosting) Salary() Salary {
	return j.salary
}

func (j *JobPosting) PostedAt() time.Time {
	return j.postedAt
}

func (j *JobPosting) Details() JobPostingDetail {
	return j.details
}
