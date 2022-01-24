const path = require("path");

module.exports = {

  devServer: {
    proxy: {
      '^/': {
        target: 'http://127.0.0.1:9000',
        ws: false,
      },
      '^/collection/monitor': {
        target: 'http://127.0.0.1:9000',
        ws: true,
        onProxyReqWs: function(request) {
          request.setHeader("Origin", "http://127.0.0.1:9000");
        },
      },
    },
  },

  pluginOptions: {
    'style-resources-loader': {
      preProcessor: 'scss',
      patterns: [
        path.resolve(__dirname, "./src/common/global.scss"),
        path.resolve(__dirname, "./src/common/style.scss"),
      ]
    }
  }
}
