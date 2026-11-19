export default defineAppConfig({
  pages: ['pages/index/index', 'pages/login/index'],
  subpackages: [
    {
      root: 'subpackages/sample',
      pages: ['index'],
    },
  ],
  window: {
    backgroundTextStyle: 'light',
    navigationBarBackgroundColor: '#ffffff',
    navigationBarTitleText: 'Go Starter 小程序',
    navigationBarTextStyle: 'black',
  },
});
