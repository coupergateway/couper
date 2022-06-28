export default {
  h1: (props) => ({
    render(h) {
      return (
        <h1 style={{ color: "tomato" }} {...props}>
          {this.$slots.default}
        </h1>
      );
    }
  })
}
