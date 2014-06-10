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
      },
      "portmapping": {
        controller: "PortMappingCtrl",
        templateUrl: 'applications/portmapping.tpl.html'
      },
      "envvars": {
        controller: "EnvVarCtrl",
        templateUrl: 'applications/envvars.tpl.html'
      },
      "component": {
        controller: "AppComponentCtrl",
        templateUrl: 'applications/component.tpl.html'
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
.controller( 'ApplicationsCtrl', function ApplicationsCtrl( $scope, $animate, Application ) {

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

  $scope.currentApp = {};
  $scope.currentComponent = {};

  $scope.selectApp = function selectApp(app) {
    app.key = app.name;
    if (app != $scope.currentApp) {
      $scope.currentApp = app;
    }
    var nonEmptyObject = app && Object.getOwnPropertyNames(app).length > 0 && app.components;    
    $scope.selectComponent(nonEmptyObject ? app.components[Object.keys(app.components)[0]] : {});
  };

  $scope.addNewComponent = function addNewComponent() {
    $scope.selectComponent({});
  };

  $scope.createNewApp = function createNewApp() {
    $scope.selectApp({});
  };


  $scope.applications = Application.query(function() {
    $scope.selectApp($scope.applications[0]);
  });

  $scope.removeApp = function removeApp(app) {
    console.log("removing app", app);
    Application.remove(app, function() {
      $scope.applications = _.filter($scope.applications, function(i) { return i.name != app.name; });
      if (app.key && app.key == $scope.currentApp.key || app.name == $scope.currentApp.name) {
        $scope.createNewApp();
      }
    });
  };


  $scope.selectComponent = function selectComponent(comp) {
    $scope.currentComponent = comp;
    $scope.currentComponent.key = comp.name;
    $scope.newPortMapping = {};
    $scope.newEnvVar = {};
  };

  $scope.removeComponent = function removeComponent(comp) {
    var app = $scope.currentApp;
    var appKey = app.key;
    app.key = app.key || app.name;
    console.log("removing component with key", comp.key || "undefined", "and name", comp.name);
    delete app.components[comp.name];
    app.components = app.components || {};
    console.log("the app to submit", app);
    Application.save(app, function() {
      if ($scope.currentComponent.name == comp.name) {
        $scope.currentComponent = {};
        $scope.newPortMapping = {};
        $scope.newEnvVar = {};
      }
    });
  };

  $scope.update = function update(app) {
    if (app.key && app.key != app.name) {
      Application.remove({name: app.key}, function() {
        Application.save(app, function(){
          app.key = app.name;
          $scope.applications = _.filter($scope.applications, function(item) { return item.name != app.name;});
          $scope.applications.push(app);
        });
      });
    } else {
      Application.save(app, function(newApp) {
        app.key = newApp.name;
      });  
    }
    
  };

  $scope.saveCurrentComponent = function saveCurrentComponent() {
    var app = $scope.currentApp;
    var appKey = app.key;
    var selected = $scope.currentComponent;
    app.key = app.key || app.name;
    console.log("updating the current application, for key", selected.key, "and name", selected.name);
    if (selected.key && selected.name != selected.key) {
      delete $scope.currentApp.components[selected.key];
    }
    app.components = app.components || {};
    app.components[selected.name] = selected;
    console.log("the app to submit", $scope.currentApp);
    Application.save(app, function(newApp) {
      $scope.currentComponent.key = selected.name;
      if (!appKey) {
        $scope.applications = _.filter($scope.applications, function(item) { return item.name != newApp.name;});
        $scope.applications.push(newApp);
      }
    });
  };

  // $scope.createTestApp = function createTestApp() {
  //   var id = $scope.applications.length;
  //   var app = new Application({
  //     name: "test-app-" + id,
  //     components: {
  //       "nginx" : {
  //         name: "nginx",
  //         cpus: 1,
  //         mem: 256,
  //         dist_url: "yum://nginx",
  //         ports: {
  //           "http": 443
  //         },
  //         version: "0.0.1",
  //         distribution: "package",
  //         component_type: "service"
  //       }
  //     }
  //   });
  //   app.components["test-app-"+id+"-service"] = {
  //     cpus: 1,
  //     mem: 256,
  //     dist_url: "hdfs://[namenode]/applications/test-app-service.jar",
  //     ports: {
  //       "http": 8000
  //     },
  //     version: "0.0.1",
  //     distribution: "fat_jar",
  //     component_type: "service"
  //   };
  //   Application.save(app, function (app){
  //     $scope.applications.push(app);
  //   });
  // };

  $scope.newPortMapping = $scope.newPortMapping || {};

  $scope.savePortMapping = function savePortMapping(scheme, port, clearNew)  {
    console.log("saving port mapping " + scheme);
    var selected = $scope.currentComponent;
    var app = _.cloneDeep($scope.currentApp);    
    app.components = app.components || {};
    selected.ports = selected.ports || {};
    app.components[selected.name].ports = app.components[selected.name].ports || {};
    app.components[selected.name].ports[scheme] = port;    

    Application.save(app, function() {

      selected.ports[scheme] = port;
      if (clearNew) {
        $scope.newPortMapping = {};
      }
    });
  };

  $scope.clearPortMapping = function clearPortMapping() {
    console.log("clearing port mapping " + angular.toJson($scope.newPortMapping, true));
    $scope.newPortMapping = {};
  };

  $scope.addPortMapping = function addPortMapping() {
    console.log("adding port mapping " + angular.toJson($scope.newPortMapping, true));
    var scheme = $scope.newPortMapping.scheme;
    var port = $scope.newPortMapping.port;

    $scope.savePortMapping(scheme, port, true);
  };

  $scope.deletePortMapping = function deletePortMapping(key) {
    console.log("deleting port mapping " + key);
    var selected = $scope.currentComponent;
    var app = _.cloneDeep($scope.currentApp);
    delete app.components[$scope.currentComponent.name].ports[key];
    Application.save(app, function(){
      delete selected.ports[key];
    });
  };

  $scope.newEnvVar = {};

  $scope.saveEnvVar = function saveEnvVar(key, value, clearNew) {
    var selected = $scope.currentComponent;
    var app = _.cloneDeep($scope.currentApp);
    console.log("saving environment variable", key, "value", value);
    app.components = app.components || {};
    selected.env = selected.env || {};
    app.components[selected.name].env = app.components[selected.name].env || {};
    app.components[selected.name].env[key] = value;
    Application.save(app, function() {
      selected.env = selected.env || {};
      selected.env[key] = value;
      if (clearNew) {
        $scope.newEnvVar = {};
      }
    });
  };

  $scope.clearEnvVar = function clearEnvVar() {
    console.log("clearing environment variable " + angular.toJson($scope.newEnvVar, true));
    $scope.newEnvVar = {};
  };

  $scope.addEnvVar = function addEnvVar() {
    console.log("adding environment variable " + angular.toJson($scope.newEnvVar, true));
    var key = $scope.newEnvVar.key;
    var value = $scope.newEnvVar.value;

    $scope.saveEnvVar(key, value, true);
  };

  $scope.deleteEnvVar = function deleteEnvVar(key) {
    console.log("deleting environment variable " + key);
    var selected = $scope.currentComponent;
    var app = _.cloneDeep($scope.currentApp);
    delete app.components[$scope.currentComponent.name].env[key];
    Application.save(app, function(){
      delete selected.env[key];
    });
  };
})

;

