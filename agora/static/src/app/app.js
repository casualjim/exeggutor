angular.module( 'agora', [
  'templates-app',
  'templates-common',
  'agora.home',
  'agora.applications',
  'agora.about',
  'ui.router',
  'ngResource'
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

;

