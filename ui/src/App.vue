<template>
  <div id="app">

    <b-container>

      <div id="nav">
        <b-navbar toggleable="lg" type="dark" variant="info">
          <b-navbar-brand to="/">Enduro</b-navbar-brand>
          <b-navbar-toggle target="nav-collapse"></b-navbar-toggle>
          <b-collapse id="nav-collapse" is-nav>
            <b-navbar-nav>
              <b-nav-item :to="{ name: 'collections' }">Collections</b-nav-item>
            </b-navbar-nav>
          </b-collapse>
        </b-navbar>
      </div>

      <router-view/>

    </b-container>

  </div>
</template>

<style lang="scss">

@import 'node_modules/bootstrap/scss/bootstrap';
@import 'node_modules/bootstrap-vue/src/index.scss';

#app {
  font-family: 'Avenir', Helvetica, Arial, sans-serif;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
  color: #2c3e50;
}

#nav {
  margin-bottom: 10px;
  .nav-link.router-link-exact-active {
    color: white;
    background-color: #0002;
  }
}

.breadcrumb {
  background-color: #fff;
  padding: 0.2rem 1rem;
}

</style>
