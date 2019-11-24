const path = require("path");

module.exports = {
  devServer: {
    proxy: 'http://localhost:9000'
  },

  pluginOptions: {
    'style-resources-loader': {
      preProcessor: 'scss',
      patterns: [
        path.resolve(__dirname, "./src/common/global.scss"),
      ]
    }
  }
}
