package public

import (
	"net/http"
	dbModel "playbook-dispatcher/internal/common/model/db"
	"playbook-dispatcher/internal/common/utils"
	"playbook-dispatcher/internal/common/utils/test"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

func listRunHosts(keysAndValues ...interface{}) (*RunHosts, *ApiRunHostsListResponse) {
	raw := listRunHostsRaw(keysAndValues...)
	res, err := ParseApiRunHostsListResponse(raw)
	Expect(err).ToNot(HaveOccurred())

	return res.JSON200, res
}

func listRunHostsRaw(keysAndValues ...interface{}) *http.Response {
	return doGet("http://localhost:9002/api/playbook-dispatcher/v1/run_hosts", keysAndValues...)
}

var _ = Describe("runHostList", func() {
	db := test.WithDatabase()

	Describe("list hosts", func() {
		It("by default returns a list of run hosts", func() {
			data := test.NewRun(accountNumber())
			data.Events = utils.MustMarshal(test.EventSequenceOk("2303e668-dff6-4e4b-8979-71ab6dd14d42", "localhost"))
			Expect(db().Create(&data).Error).ToNot(HaveOccurred())

			runs, res := listRunHosts()
			Expect(res.StatusCode()).To(Equal(http.StatusOK))
			Expect(runs.Data).To(HaveLen(1))
			Expect(*runs.Data[0].Host).To(Equal("localhost"))
			Expect(string(*runs.Data[0].Status)).To(Equal("running"))
			Expect(string(*runs.Data[0].Run.Id)).To(Equal(data.ID.String()))
		})

		Describe("filtering", func() {
			It("filters by host status", func() {
				data := []dbModel.Run{
					test.NewRunWithStatus(accountNumber(), "success"),
					test.NewRunWithStatus(accountNumber(), "failure"),
				}

				data[0].Events = utils.MustMarshal(test.EventSequenceOk("ee44fcba-60d2-4a2a-a6bf-74875487c9dc", "localhost"))
				data[1].Events = utils.MustMarshal(test.EventSequenceOk("aea95ec9-4db6-4756-b10e-12bf42444ace", "localhost"))
				Expect(db().Create(&data).Error).ToNot(HaveOccurred())

				runs, res := listRunHosts("filter[status]", "failure")
				Expect(res.StatusCode()).To(Equal(http.StatusOK))
				Expect(runs.Data).To(HaveLen(1))
				Expect(string(*runs.Data[0].Run.Id)).To(Equal(data[1].ID.String()))
			})

			It("filters by run id", func() {
				data := []dbModel.Run{
					test.NewRun(accountNumber()),
					test.NewRun(accountNumber()),
					test.NewRun(accountNumber()),
				}

				data[0].Events = utils.MustMarshal(test.EventSequenceOk("ee44fcba-60d2-4a2a-a6bf-74875487c9dc", "localhost"))
				data[1].Events = utils.MustMarshal(test.EventSequenceOk("aea95ec9-4db6-4756-b10e-12bf42444ace", "localhost"))
				data[2].Events = utils.MustMarshal(test.EventSequenceOk("25e32ee0-41e5-4e14-a63b-35e58d024963", "localhost"))
				Expect(db().Create(&data).Error).ToNot(HaveOccurred())

				runs, res := listRunHosts("filter[run][id]", data[1].ID.String())
				Expect(res.StatusCode()).To(Equal(http.StatusOK))
				Expect(runs.Data).To(HaveLen(1))
				Expect(string(*runs.Data[0].Run.Id)).To(Equal(data[1].ID.String()))
			})

			It("filters by run labels", func() {
				data := []dbModel.Run{
					test.NewRun(accountNumber()),
					test.NewRun(accountNumber()),
					test.NewRun(accountNumber()),
				}

				data[0].Labels = map[string]string{"remediation": "0"}
				data[1].Labels = map[string]string{"remediation": "1"}
				data[2].Labels = map[string]string{"remediation": "2"}

				data[0].Events = utils.MustMarshal(test.EventSequenceOk("ee44fcba-60d2-4a2a-a6bf-74875487c9dc", "localhost"))
				data[1].Events = utils.MustMarshal(test.EventSequenceOk("aea95ec9-4db6-4756-b10e-12bf42444ace", "localhost"))
				data[2].Events = utils.MustMarshal(test.EventSequenceOk("25e32ee0-41e5-4e14-a63b-35e58d024963", "localhost"))
				Expect(db().Create(&data).Error).ToNot(HaveOccurred())

				runs, res := listRunHosts("filter[run][labels][remediation]", "2")
				Expect(res.StatusCode()).To(Equal(http.StatusOK))
				Expect(runs.Data).To(HaveLen(1))
				Expect(string(*runs.Data[0].Run.Id)).To(Equal(data[2].ID.String()))
			})
		})
	})

	Describe("sparse fieldsets", func() {
		BeforeEach(func() {
			data := test.NewRun(accountNumber())
			data.Events = utils.MustMarshal(test.EventSequenceOk("ee44fcba-60d2-4a2a-a6bf-74875487c9dc", "localhost"))
			Expect(db().Create(&data).Error).ToNot(HaveOccurred())
		})

		DescribeTable("happy path", fieldTester(listRunHostsRaw),
			Entry("single field", "host"),
			Entry("defaults defined explicitly", "host", "status", "run"),
			Entry("all fields", "host", "status", "run", "stdout"),
		)

		It("400s on invalid value", func() {
			raw := listRunHostsRaw("fields[data]", "host,salad")
			Expect(raw.StatusCode).To(Equal(http.StatusBadRequest))
			res, err := ParseApiRunHostsListResponse(raw)
			Expect(err).ToNot(HaveOccurred())
			Expect(res.JSON400.Message).To(Equal("unknown field: salad"))
		})
	})

	DescribeTable("pagination",
		func(expected, limit, offset int) {
			Expect(db().Create(test.NewRunsWithLocalhost(accountNumber(), 11)).Error).ToNot(HaveOccurred())

			runs, res := listRunHosts("limit", limit, "offset", offset)
			Expect(res.StatusCode()).To(Equal(http.StatusOK))
			Expect(runs.Data).To(HaveLen(expected))
		},

		Entry("limit=2", 2, 2, 0),
		Entry("limit=5", 5, 5, 0),
		Entry("limit=5, offset=10", 1, 5, 10),
		Entry("limit=5, offset=20", 0, 5, 20),
	)
})
