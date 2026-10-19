package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"job-tracker/backend/internal/models"
)

type dummyApp struct {
	company     string
	role        string
	location    string
	jobURL      string
	salary      string
	notes       string
	description string
	history     []dummyEvent
}

type dummyEvent struct {
	status   models.Status
	notes    string
	daysAgo  int
	hoursAgo int
}

// SeedDummyData populates the database with realistic sample job applications in Switzerland
// using CHF monetary units and complete status progression history.
// It is a no-op if the applications table already contains records.
func SeedDummyData(ctx context.Context, conn *sql.DB) error {
	var count int
	if err := conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM applications").Scan(&count); err != nil {
		return fmt.Errorf("count applications: %w", err)
	}
	if count > 0 {
		return nil
	}

	now := time.Now().UTC()

	dummies := []dummyApp{
		{
			company:     "GigaCorp",
			role:        "Senior Site Reliability Engineer",
			location:    "Zurich, Switzerland",
			jobURL:      "https://careers.gigacorp.example/jobs/123456",
			salary:      "CHF 210'000 – CHF 260'000",
			notes:       "Referral from former colleague. Focus on distributed infrastructure & Kubernetes reliability.",
			description: "About the team:\nGigaCorp's Site Reliability Engineering (SRE) team in Zurich creates, fixes, and enhances distributed systems to keep large-scale cloud services running smoothly.\n\nKey Responsibilities:\n- Design, deploy, and maintain large-scale distributed architectures.\n- Improve system observability, SLIs/SLOs, and automated failure recovery.\n- Participate in on-call rotations and lead blameless post-mortems.\n- Collaborate with software engineers to optimize latency and throughput.\n\nQualifications:\n- 5+ years experience building and operating production distributed systems.\n- Strong proficiency in Go, C++, or Python.\n- Deep expertise in Linux internals, networking protocols, and container orchestration.",
			history: []dummyEvent{
				{status: models.StatusSaved, notes: "Saved job posting from professional network", daysAgo: 30},
				{status: models.StatusApplied, notes: "Submitted application via referral", daysAgo: 28},
				{status: models.StatusResponseReceived, notes: "Recruiter contacted to schedule initial phone screen", daysAgo: 24},
				{status: models.StatusInterviewing, notes: "Completed technical screen and 4-round virtual onsite", daysAgo: 16},
				{status: models.StatusOffer, notes: "Received formal offer letter! Base CHF 220k + CHF 35k bonus + equity", daysAgo: 4},
			},
		},
		{
			company:     "Neutron Privacy",
			role:        "Senior Backend Engineer (Go)",
			location:    "Geneva, Switzerland (Hybrid)",
			jobURL:      "https://neutron.example/careers/senior-backend-engineer",
			salary:      "CHF 145'000 – CHF 170'000",
			notes:       "Privacy-first encrypted services. Tech stack: Go, PostgreSQL, Redis, Kubernetes.",
			description: "About Neutron:\nNeutron is a Swiss technology company providing privacy-first online services, including encrypted communications, VPN, and zero-knowledge storage.\n\nWhat You'll Do:\n- Architect, build, and maintain highly scalable backend services in Go.\n- Ensure end-to-end security and cryptographic integrity of user data.\n- Optimize high-throughput APIs serving millions of active users.\n- Contribute to architectural discussions and code reviews.\n\nWhat You Bring:\n- 5+ years of software development experience with strong Go knowledge.\n- Solid background in concurrent programming, SQL optimization, and distributed caching.\n- Familiarity with zero-knowledge encryption and cryptographic protocols is an asset.\n- Based in Switzerland or willing to relocate to Geneva.",
			history: []dummyEvent{
				{status: models.StatusApplied, notes: "Applied directly on Neutron careers page", daysAgo: 21},
				{status: models.StatusResponseReceived, notes: "Engineering manager reached out for screening", daysAgo: 14},
				{status: models.StatusInterviewing, notes: "System design & live coding round completed", daysAgo: 6},
			},
		},
		{
			company:     "AlpineCom",
			role:        "Cloud Solutions Architect",
			location:    "Bern, Switzerland",
			jobURL:      "https://jobs.alpinecom.example/cloud-architect",
			salary:      "CHF 150'000 – CHF 180'000",
			notes:       "Enterprise cloud transition team. Great Swiss pension and benefits package.",
			description: "As a Cloud Solutions Architect at AlpineCom, you will guide major Swiss enterprises in designing scalable, resilient multi-cloud architectures.\n\nResponsibilities:\n- Advise enterprise customers on cloud migration strategies (AWS, Azure, Swiss Private Cloud).\n- Design infrastructure as code templates using Terraform and Helm.\n- Establish zero-trust security postures and compliance frameworks (FINMA, Swiss FADP).\n- Mentor junior engineers and conduct technical workshops.\n\nRequirements:\n- Proven experience as a Cloud Architect or Principal Cloud Engineer.\n- Deep knowledge of Kubernetes, Terraform, and cloud networking.\n- Fluent in English and German.",
			history: []dummyEvent{
				{status: models.StatusApplied, notes: "Application submitted online", daysAgo: 38},
				{status: models.StatusResponseReceived, notes: "HR phone screening", daysAgo: 32},
				{status: models.StatusInterviewing, notes: "Architecture presentation round completed", daysAgo: 22},
				{status: models.StatusOffer, notes: "Received formal offer for CHF 165'000", daysAgo: 12},
				{status: models.StatusAccepted, notes: "Signed and returned the employment contract!", daysAgo: 7},
			},
		},
		{
			company:     "BobbyTech",
			role:        "Lead Embedded Software Engineer",
			location:    "Lausanne, Switzerland",
			jobURL:      "https://jobs.bobbytech.example/embedded-lead",
			salary:      "CHF 155'000 – CHF 185'000",
			notes:       "Workspace solutions division at EPFL Innovation Park.",
			description: "Lead the firmware architecture for BobbyTech's next-generation smart conference room hardware.\n\nKey Responsibilities:\n- Design low-latency audio/video embedded firmware running on ARM Cortex processors.\n- Optimize Bluetooth Low Energy (BLE) and USB connectivity stacks.\n- Collaborate closely with hardware, RF, and industrial design teams.\n\nRequirements:\n- 7+ years of embedded C/C++ development with RTOS (FreeRTOS or Zephyr).\n- Experience with low-power wireless communication and hardware debugging.",
			history: []dummyEvent{
				{status: models.StatusApplied, notes: "Applied via BobbyTech careers", daysAgo: 16},
				{status: models.StatusResponseReceived, notes: "Screening call with talent acquisition partner", daysAgo: 5},
			},
		},
		{
			company:     "BigCorporateBank",
			role:        "Principal Platform Engineer",
			location:    "Zurich, Switzerland",
			jobURL:      "https://jobs.bigcorporatebank.example/platform-engineer",
			salary:      "CHF 170'000 – CHF 200'000",
			notes:       "Internal developer platform engineering team.",
			description: "BigCorporateBank is seeking a Principal Platform Engineer to modernize internal CI/CD pipelines and developer tooling.\n\nResponsibilities:\n- Build developer self-service portals and automated compliance scanning.\n- Manage enterprise Kubernetes clusters and GitOps workflows.\n- Enhance build caching and test execution speeds for 10,000+ developers.\n\nRequirements:\n- Extensive experience in Go, Python, Kubernetes, and ArgoCD.\n- Experience operating within regulated financial sector environments.",
			history: []dummyEvent{
				{status: models.StatusApplied, notes: "Applied via internal referral", daysAgo: 26},
				{status: models.StatusRejected, notes: "Position put on freeze / candidate pool closed", daysAgo: 9},
			},
		},
		{
			company:     "Stride Athletics",
			role:        "Full Stack Engineer (React & Go)",
			location:    "Zurich, Switzerland",
			jobURL:      "https://stride-athletics.example/careers/full-stack",
			salary:      "CHF 130'000 – CHF 155'000",
			notes:       "Fast growing Swiss sportswear innovator. Office in Zurich West.",
			description: "Join Stride's digital team building responsive e-commerce and member experiences for runners globally.\n\nResponsibilities:\n- Build modern web apps using React, Next.js, and TypeScript.\n- Develop performant microservices in Go backed by PostgreSQL.\n- Optimize page performance and internationalization.\n\nRequirements:\n- 3+ years experience across modern web technologies (React/Go).\n- Passion for sports, active lifestyle, and clean code.",
			history: []dummyEvent{
				{status: models.StatusApplied, notes: "Applied through Stride career portal", daysAgo: 3},
			},
		},
		{
			company:     "CleanSyntax",
			role:        "Static Code Analysis Engineer",
			location:    "Geneva, Switzerland",
			jobURL:      "https://www.cleansyntax.example/jobs/compiler-engineer",
			salary:      "CHF 140'000 – CHF 165'000",
			notes:       "Makers of automated code quality and security analysis developer tools.",
			description: "Develop language analyzers and compiler frontend tools detecting security flaws and code quality issues.\n\nRequirements:\n- Strong knowledge of compilers, abstract syntax trees (ASTs), and control flow graphs.\n- Proficiency in Java, Go, C++, or Rust.\n- Passion for software engineering craftsmanship.",
			history: []dummyEvent{
				{status: models.StatusSaved, notes: "Job bookmarked to review requirements and prepare cover letter", daysAgo: 11},
			},
		},
		{
			company:     "Joshe",
			role:        "Senior Bio-Informatics Data Engineer",
			location:    "Basel, Switzerland",
			jobURL:      "https://careers.joshe.example/data-engineer",
			salary:      "CHF 160'000 – CHF 190'000",
			notes:       "Genomics data pipeline modernization.",
			description: "Build petabyte-scale genomics processing pipelines supporting personalized healthcare research.\n\nRequirements:\n- Experience with distributed computing frameworks (Spark, Ray, Nextflow).\n- Strong programming skills in Python and Go.\n- Familiarity with cloud data lakes (AWS S3, Snowflake).",
			history: []dummyEvent{
				{status: models.StatusApplied, notes: "Applied via corporate portal", daysAgo: 48},
				{status: models.StatusGhosted, notes: "No response received after 30+ days", daysAgo: 15},
			},
		},
		{
			company:     "Apex Automation",
			role:        "Industrial IoT Software Engineer",
			location:    "Baden / Zurich, Switzerland",
			jobURL:      "https://careers.apex-automation.example/iot-software",
			salary:      "CHF 135'000 – CHF 155'000",
			notes:       "Industrial automation telemetry division.",
			description: "Develop edge telemetry gateways connecting industrial robotics to cloud monitoring platforms.\n\nRequirements:\n- Embedded Linux, Go, MQTT, and Docker.\n- Experience with real-time industrial protocols (OPC-UA, Modbus).",
			history: []dummyEvent{
				{status: models.StatusApplied, notes: "Applied through Apex careers", daysAgo: 30},
				{status: models.StatusResponseReceived, notes: "Phone interview with hiring manager", daysAgo: 21},
				{status: models.StatusWithdrawn, notes: "Withdrew application after accepting another offer", daysAgo: 10},
			},
		},
	}

	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin seed tx: %w", err)
	}
	defer tx.Rollback()

	for _, app := range dummies {
		firstEvent := app.history[0]
		createdAt := now.AddDate(0, 0, -firstEvent.daysAgo).Add(time.Duration(-firstEvent.hoursAgo) * time.Hour).Format("2006-01-02 15:04:05")

		res, err := tx.ExecContext(ctx, `
			INSERT INTO applications (company_name, role_title, location, job_url, salary_range, notes, job_description, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			app.company, app.role, app.location, app.jobURL, app.salary, app.notes, app.description, createdAt)
		if err != nil {
			return fmt.Errorf("insert dummy app %s: %w", app.company, err)
		}
		appID, err := res.LastInsertId()
		if err != nil {
			return fmt.Errorf("get dummy app id: %w", err)
		}

		for _, evt := range app.history {
			changedAt := now.AddDate(0, 0, -evt.daysAgo).Add(time.Duration(-evt.hoursAgo) * time.Hour).Format("2006-01-02 15:04:05")
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO application_status_history (application_id, status, notes, changed_at)
				VALUES (?, ?, ?, ?)`,
				appID, evt.status, evt.notes, changedAt); err != nil {
				return fmt.Errorf("insert dummy history for %s: %w", app.company, err)
			}
		}
	}

	return tx.Commit()
}
