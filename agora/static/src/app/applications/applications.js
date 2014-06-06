angular.module( 'agora.applications', [
  'ui.router',
  'ngResource'
])

.config(function config( $stateProvider ) {
  $stateProvider.state( 'applications', {
    url: '/applications',
    views: {
      "main": {
        controller: 'ApplicationsCtrl',
        templateUrl: 'applications/applications.tpl.html'
      }
    },
    data:{ pageTitle: 'Manage Applications' }
  });
})


.factory("Application", ['$resource', function ($resource) {
  return $resource("/api/applications/:name");
}])


/**
 * And of course we define a controller for our route.
 */
.controller( 'ApplicationsCtrl', function ApplicationsCtrl( $scope, $http, Application ) {

  

  $scope.distributions = {
    "package": "Package (RPM/DEB)",
    "docker": "Docker container",
    "script": "Single binary",
    "fat_jar": "Fat Jar/War"
  };

  $scope.componentTypes = {
    "service": "Service",
    "task": "One-Off job",
    "cron": "CRON job",
    "spark": "Spark Job"
  };

  $scope.applications = Application.query();

  $scope.currentApp = null;

  $scope.selectApp = function selectApp(app) {
    $scope.currentApp = app;
    $scope.currentComponent = app.components[0];
  };


  $scope.selectComponent = function selectComponent(comp) {
    $scope.currentComponent = comp;
  };


  $scope.update = function updateApp(app) {
    console.log("updating app: " + app.name);
  };

  $scope.createTestApp = function createTestApp() {
    var app = new Application({
      name: "test-app",
      components: [
        {
          name: "test-app-service",
          cpus: 1,
          mem: 256,
          dist_url: "hdfs://[namenode]/applications/test-app-service.jar",
          ports: {
            "http": 8000
          },
          version: "0.0.1",
          distribution: "fat_jar",
          component_type: "service"
        }
      ]
    });
    Application.save(app);
  };


})

;

