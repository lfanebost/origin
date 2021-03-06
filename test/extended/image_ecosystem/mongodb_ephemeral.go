package image_ecosystem

import (
	"fmt"

	g "github.com/onsi/ginkgo"
	o "github.com/onsi/gomega"

	"time"

	exutil "github.com/openshift/origin/test/extended/util"
	"github.com/openshift/origin/test/extended/util/db"
)

var _ = g.Describe("[sig-devex][Feature:ImageEcosystem][mongodb] openshift mongodb image", func() {
	defer g.GinkgoRecover()

	templatePath := "mongodb-ephemeral"
	oc := exutil.NewCLI("mongodb-create").Verbose()

	g.Context("", func() {
		g.BeforeEach(func() {
			exutil.PreTestDump()
		})

		g.AfterEach(func() {
			if g.CurrentGinkgoTestDescription().Failed {
				exutil.DumpPodStates(oc)
				exutil.DumpPodLogsStartingWith("", oc)
			}
		})

		g.Describe("creating from a template", func() {
			g.It(fmt.Sprintf("should instantiate the template"), func() {

				exutil.WaitForOpenShiftNamespaceImageStreams(oc)
				g.By("creating a new app")
				o.Expect(oc.Run("new-app").Args(templatePath).Execute()).Should(o.Succeed())

				g.By("waiting for the deployment to complete")
				err := exutil.WaitForDeploymentConfig(oc.KubeClient(), oc.AppsClient().AppsV1(), oc.Namespace(), "mongodb", 1, true, oc)
				o.Expect(err).ShouldNot(o.HaveOccurred())

				g.By("expecting the mongodb pod is running")
				podNames, err := exutil.WaitForPods(
					oc.KubeClient().CoreV1().Pods(oc.Namespace()),
					exutil.ParseLabelsOrDie("name=mongodb"),
					exutil.CheckPodIsRunning,
					1,
					4*time.Minute,
				)
				o.Expect(err).ShouldNot(o.HaveOccurred())
				o.Expect(podNames).Should(o.HaveLen(1))

				g.By("expecting the mongodb service is answering for ping")
				mongo := db.NewMongoDB(podNames[0])
				ok, err := mongo.IsReady(oc)
				o.Expect(err).ShouldNot(o.HaveOccurred())
				o.Expect(ok).Should(o.BeTrue())

				g.By("expecting that we can insert a new record")
				result, err := mongo.Query(oc, `db.foo.save({ "status": "passed" })`)
				o.Expect(err).ShouldNot(o.HaveOccurred())
				o.Expect(result).Should(o.ContainSubstring(`WriteResult({ "nInserted" : 1 })`))

				g.By("expecting that we can read a record")
				findCmd := "printjson(db.foo.find({}, {_id: 0}).toArray())" // don't include _id field to output because it changes every time
				result, err = mongo.Query(oc, findCmd)
				o.Expect(err).ShouldNot(o.HaveOccurred())
				o.Expect(result).Should(o.ContainSubstring(`{ "status" : "passed" }`))
			})
		})
	})
})
