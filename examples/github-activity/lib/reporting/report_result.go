package reporting

type Report struct {
	Snapshot ReportingSnapshot
	Summary  Summary
}

type Result struct {
	Report            Report
	ActivityCount     int
	OrganizationCount int
	Err               error
}
