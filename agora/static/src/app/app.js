angular.module( 'agora', [
  'templates-app',
  'templates-common',
  'agora.home',
  'agora.applications',
  'agora.about',
  'ui.router',
  'ngResource',
  'ngAnimate-animate.css'
])

.config( function myAppConfig ( $stateProvider, $urlRouterProvider ) {
  $urlRouterProvider.otherwise( '/home' );
})

.run( function run ($http) {
  $http.defaults.headers.common['Content-Type'] = 'application/json;charset=utf8';
})

.controller( 'AppCtrl', function AppCtrl ( $scope, $location ) {
  $scope.$on('$stateChangeSuccess', function(event, toState, toParams, fromState, fromParams){
    if ( angular.isDefined( toState.data.pageTitle ) ) {
      $scope.pageTitle = toState.data.pageTitle + ' | Agora' ;
    }
  });
})


.directive( 'agoraAutosave', function ($timeout) {
  return {
    restrict: 'A',
    link: function (scope, element, attrs) {
      var expr = attrs.agoraAutosave;
      var frm = attrs.agoraForm;
      var model = attrs.ngModel;

      var previous = null;
      var tio = null;

      var performSave = function() {
        // console.log("performing autosave, the form", scope[frm], "the validity", scope[frm] && scope[frm].$valid);
        var canSave = scope[frm] && scope[frm].$valid;
        // console.log("autosave possible", canSave, "values differ", previous != scope.value);
        if (canSave) {
          if (previous != scope.value) {
            // console.log("actually invoking save on controller");
            scope.$apply(expr);
          }
          previous = null;
          if (tio) {
            $timeout.cancel(tio);
            tio = null;
          }
        }
      };

      // debounce updates, so when the user stops typing we bgsave his changes
      scope.$watch(model, function(newVal, oldVal) {
        // console.log("the values are different", newVal != oldVal);
        if (newVal != oldVal) {
          previous = oldVal;
          if (tio) {
            $timeout.cancel(tio);
          }
          tio = $timeout(performSave, 1000);
        }
      });

      // Save the previous value on focus, best effort change detection
      element.on('focus', function() {
        var canSave = scope[frm].$valid;
        if (canSave) {
          previous = scope.value;
        }
      });

      // on blur we can assume the user is done.
      element.on('blur', performSave);
    }
  };
})

;
