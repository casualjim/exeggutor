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
  };


  $scope.update = function updateApp(app) {
    console.log("updating app: " + app.name);
  };



})

;

